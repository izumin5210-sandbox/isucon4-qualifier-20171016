#!/bin/bash

go get github.com/gin-gonic/gin
go get github.com/gin-contrib/sessions
go get github.com/go-sql-driver/mysql
go get github.com/garyburd/redigo/redis
go build -o golang-webapp .
