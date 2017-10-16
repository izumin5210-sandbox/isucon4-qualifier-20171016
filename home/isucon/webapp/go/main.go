package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB
var pool *redis.Pool
var (
	UserLockThreshold int
	IPBanThreshold    int
)

func init() {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local",
		getEnv("ISU4_DB_USER", "root"),
		getEnv("ISU4_DB_PASSWORD", ""),
		getEnv("ISU4_DB_HOST", "localhost"),
		getEnv("ISU4_DB_PORT", "3306"),
		getEnv("ISU4_DB_NAME", "isu4_qualifier"),
	)

	var err error

	db, err = sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}

	pool = &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 10 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", ":6379")
		},
	}

	UserLockThreshold, err = strconv.Atoi(getEnv("ISU4_USER_LOCK_THRESHOLD", "3"))
	if err != nil {
		panic(err)
	}

	IPBanThreshold, err = strconv.Atoi(getEnv("ISU4_IP_BAN_THRESHOLD", "10"))
	if err != nil {
		panic(err)
	}
}

func main() {
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	store := sessions.NewCookieStore([]byte("secret-isucon"))
	r.Use(sessions.Sessions("isucon_go_session", store))

	r.GET("/", func(c *gin.Context) {
		session := sessions.Default(c)
		c.HTML(200, "index.tmpl", gin.H{"Flash": getFlash(session, "notice")})
	})

	r.POST("/login", func(c *gin.Context) {
		session := sessions.Default(c)
		defer session.Save()
		user, err := attemptLogin(c.Request)

		notice := ""
		if err != nil || user == nil {
			switch err {
			case ErrBannedIP:
				notice = "You're banned."
			case ErrLockedUser:
				notice = "This account is locked."
			default:
				notice = "Wrong username or password"
			}

			session.Set("notice", notice)
			c.Redirect(302, "/")
			return
		}

		session.Set("user_id", strconv.Itoa(user.ID))
		c.Redirect(302, "/mypage")
	})

	r.GET("/mypage", func(c *gin.Context) {
		session := sessions.Default(c)
		defer session.Save()
		currentUser := getCurrentUser(session.Get("user_id"))

		if currentUser == nil {
			session.Set("notice", "You must be logged in")
			c.Redirect(302, "/")
			return
		}

		currentUser.getLastLogin()
		c.HTML(200, "mypage.tmpl", currentUser)
	})

	r.GET("/report", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"banned_ips":   bannedIPs(),
			"locked_users": lockedUsers(),
		})
	})

	http.ListenAndServe(":8080", r)
}
