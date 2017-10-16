package main

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/garyburd/redigo/redis"
)

var (
	ErrBannedIP      = errors.New("Banned IP")
	ErrLockedUser    = errors.New("Locked user")
	ErrUserNotFound  = errors.New("Not found user")
	ErrWrongPassword = errors.New("Wrong password")
)

const (
	KeyLoginFailureCountByUserID = "login_failure_count:user_id"
	KeyLoginFailureCountByIP     = "login_failure_count:ip"
)

func createLoginLog(succeeded bool, remoteAddr, login string, user *User) error {
	conn := pool.Get()
	defer conn.Close()

	var errs []error

	if succeeded {
		_, err := conn.Do("HSET", KeyLoginFailureCountByIP, remoteAddr, 0)
		if err != nil {
			errs = append(errs, err)
		}
		if user != nil {
			_, err := conn.Do("HSET", KeyLoginFailureCountByUserID, user.ID, 0)
			if err != nil {
				errs = append(errs, err)
			}
			_, err = conn.Do("HSET", user.getLastLoginKey(), KeyLastLoginIP, remoteAddr)
			if err != nil {
				errs = append(errs, err)
			}
			_, err = conn.Do("HSET", user.getLastLoginKey(), KeyLastLoginTime, time.Now().Format("2006-01-02 15:04:05"))
			if err != nil {
				errs = append(errs, err)
			}
		}
	} else {
		_, err := conn.Do("HINCRBY", KeyLoginFailureCountByIP, remoteAddr, 1)
		if err != nil {
			errs = append(errs, err)
		}
		if user != nil {
			_, err := conn.Do("HINCRBY", KeyLoginFailureCountByUserID, user.ID, 1)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

func isLockedUser(user *User) (bool, error) {
	if user == nil {
		return false, nil
	}

	conn := pool.Get()
	defer conn.Close()
	cnt, err := redis.Int(conn.Do("HGET", KeyLoginFailureCountByUserID, user.ID))

	switch {
	case err == redis.ErrNil:
		return false, nil
	case err != nil:
		return false, err
	}

	return UserLockThreshold <= cnt, nil
}

func isBannedIP(ip string) (bool, error) {
	conn := pool.Get()
	defer conn.Close()
	cnt, err := redis.Int(conn.Do("HGET", KeyLoginFailureCountByIP, ip))

	switch {
	case err == redis.ErrNil:
		return false, nil
	case err != nil:
		return false, err
	}

	return IPBanThreshold <= cnt, nil
}

func attemptLogin(req *http.Request) (*User, error) {
	succeeded := false
	user := &User{}

	loginName := req.PostFormValue("login")
	password := req.PostFormValue("password")

	remoteAddr := req.RemoteAddr
	if xForwardedFor := req.Header.Get("X-Forwarded-For"); len(xForwardedFor) > 0 {
		remoteAddr = xForwardedFor
	}

	defer func() {
		createLoginLog(succeeded, remoteAddr, loginName, user)
	}()

	row := db.QueryRow(
		"SELECT id, login, password_hash, salt FROM users WHERE login = ?",
		loginName,
	)
	err := row.Scan(&user.ID, &user.Login, &user.PasswordHash, &user.Salt)

	switch {
	case err == sql.ErrNoRows:
		user = nil
	case err != nil:
		return nil, err
	}

	if banned, _ := isBannedIP(remoteAddr); banned {
		return nil, ErrBannedIP
	}

	if locked, _ := isLockedUser(user); locked {
		return nil, ErrLockedUser
	}

	if user == nil {
		return nil, ErrUserNotFound
	}

	if user.PasswordHash != calcPassHash(password, user.Salt) {
		return nil, ErrWrongPassword
	}

	succeeded = true
	return user, nil
}

func getCurrentUser(userId interface{}) *User {
	user := &User{}
	row := db.QueryRow(
		"SELECT id, login, password_hash, salt FROM users WHERE id = ?",
		userId,
	)
	err := row.Scan(&user.ID, &user.Login, &user.PasswordHash, &user.Salt)

	if err != nil {
		return nil
	}

	return user
}

func bannedIPs() []string {
	ips := []string{}

	conn := pool.Get()
	defer conn.Close()

	cntByIP, err := redis.IntMap(conn.Do("HGETALL", KeyLoginFailureCountByIP))

	if err != nil {
		return ips
	}

	for ip, cnt := range cntByIP {
		if IPBanThreshold <= cnt {
			ips = append(ips, ip)
		}
	}

	return ips
}

func lockedUsers() []string {
	userIDs := []string{}

	conn := pool.Get()
	defer conn.Close()

	cntByUserID, err := redis.IntMap(conn.Do("HGETALL", KeyLoginFailureCountByUserID))

	if err != nil {
		return userIDs
	}

	for userID, cnt := range cntByUserID {
		if UserLockThreshold <= cnt {
			userIDs = append(userIDs, userID)
		}
	}

	return userIDs
}
