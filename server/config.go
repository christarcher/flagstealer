package main

import (
	"flag"
	"log"
	"time"
)

var (
	username = flag.String("u", "0rays", "用户名")
	password = flag.String("p", "", "密码")
	port     = flag.String("P", "26666", "监听端口")
)

const (
	maxFileSize  = 10 * 1024 * 1024
	dialTimeout  = 10 * time.Second
	writeTimeout = 10 * time.Second
	readTimeout  = 10 * time.Second
)

func getConfigValue(key string) string {
	var value string
	err := db.QueryRow("SELECT value FROM config WHERE key = ?", key).Scan(&value)
	if err != nil {
		log.Printf("获取配置失败 %s: %v", key, err)
		return ""
	}
	return value
}

func setConfigValue(key, value string) error {
	_, err := db.Exec("INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)", key, value)
	return err
}
