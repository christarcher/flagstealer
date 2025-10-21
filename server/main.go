package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"time"
)

//go:embed static/*
var staticFiles embed.FS

func main() {
	flag.Parse()

	if *password == "" {
		log.Fatal("需要 -p 指定密码")
	}

	if _, err := strconv.Atoi(*port); err != nil {
		log.Fatalf("端口错误: %s", *port)
	}

	if err := initDB(); err != nil {
		log.Fatalf("加载数据库错误: %v", err)
	}
	defer db.Close()

	subFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()

	// 静态文件
	mux.Handle("/", http.FileServer(http.FS(subFS)))

	// C2客户端接口
	mux.HandleFunc("/api/c2/submit-flag", submitFlag)
	mux.HandleFunc("/api/c2/heartbeat", heartbeat)
	mux.HandleFunc("/api/c2/get-rs", getReverseShell)

	// EDR客户端接口
	mux.HandleFunc("/api/agent/edr-alert", edrAlert)
	mux.HandleFunc("/api/agent/edr-suspicious-file", edrSuspiciousFile)

	// 浏览器管理接口(需要认证)
	mux.HandleFunc("/api/browser/messages", basicAuth(getMessages))
	mux.HandleFunc("/api/browser/clearmessage", basicAuth(clearMessages))
	mux.HandleFunc("/api/browser/getclients", basicAuth(getClients))
	mux.HandleFunc("/api/browser/set-client", basicAuth(setClient))
	mux.HandleFunc("/api/browser/set-rs", basicAuth(setReverseShell))
	mux.HandleFunc("/api/browser/set-flag-api", basicAuth(setFlagAPI))
	mux.HandleFunc("/api/browser/get-template", basicAuth(getTemplate))
	mux.HandleFunc("/api/browser/edit-template", basicAuth(editTemplate))

	srv := &http.Server{
		Addr:              ":" + *port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	logo := `   ___  _____        __     _______         __          _______
  / _ \|  __ \     /\\ \   / / ____|       /\ \        / /  __ \
 | | | | |__) |   /  \\ \_/ / (___ ______ /  \ \  /\  / /| |  | |
 | | | |  _  /   / /\ \\   / \___ \______/ /\ \ \/  \/ / | |  | |
 | |_| | | \ \  / ____ \| |  ____) |    / ____ \  /\  /  | |__| |
  \___/|_|  \_\/_/    \_\_| |_____/    /_/    \_\/  \/   |_____/

                                                                 `
	fmt.Println(logo)
	log.Printf("AWD消息中台启动，监听端口 %s", *port)
	log.Printf("认证信息 - 用户名: %s", *username)
	log.Printf("EDR文件大小限制: %d MB", maxFileSize/(1024*1024))

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
