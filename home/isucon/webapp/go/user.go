package main

import (
	"fmt"

	"github.com/garyburd/redigo/redis"
)

type User struct {
	ID           int
	Login        string
	PasswordHash string
	Salt         string

	LastLogin *LastLogin
}

type LastLogin struct {
	Login     string
	IP        string
	CreatedAt string
}

const (
	KeyLastLoginIP   = "ip"
	KeyLastLoginTime = "time"
)

func (u *User) getLastLoginKey() string {
	return fmt.Sprintf("last_login:%d", u.ID)
}

func (u *User) getLastLogin() *LastLogin {
	conn := pool.Get()
	defer conn.Close()

	m, err := redis.StringMap(conn.Do("HGETALL", u.getLastLoginKey()))

	if err != nil {
		return nil
	}

	u.LastLogin = &LastLogin{}
	u.LastLogin.Login = u.Login
	u.LastLogin.IP = m[KeyLastLoginIP]
	u.LastLogin.CreatedAt = m[KeyLastLoginTime]

	return u.LastLogin
}
