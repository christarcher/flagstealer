package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func edrAlert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	alertType := r.URL.Query().Get("type")
	message := r.URL.Query().Get("message")

	if alertType == "" || message == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("type and message parameters are required"))
		return
	}

	if alertType != "warning" && alertType != "info" && alertType != "success" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("type must be warning, info, or success"))
		return
	}

	clientIP := getClientIP(r)
	title := "EDR消息"
	content := fmt.Sprintf("来源: %s, 消息: %s", clientIP, message)

	addMessage(alertType, title, content)
	log.Printf("EDR告警 %s: [%s] %s", clientIP, alertType, message)

	w.WriteHeader(http.StatusOK)
}

func edrSuspiciousFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var suspFile SuspiciousFile
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&suspFile); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if suspFile.Filename == "" || suspFile.Path == "" || suspFile.Content == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("filename, path, and content are required"))
		return
	}

	filename := filepath.Base(suspFile.Filename)
	if filename == "" || filename == "." || filename == ".." {
		log.Printf("文件名不合法: %s", suspFile.Filename)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid filename"))
		return
	}

	if len(suspFile.Content) > maxFileSize*4/3 {
		log.Printf("文件过大: %d bytes (base64)", len(suspFile.Content))
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		w.Write([]byte("File too large"))
		return
	}

	content, err := base64.StdEncoding.DecodeString(suspFile.Content)
	if err != nil {
		log.Printf("base64解码失败: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid base64 content"))
		return
	}

	if len(content) > maxFileSize {
		log.Printf("解码后的文件太大: %d bytes", len(content))
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		w.Write([]byte("Decoded file too large"))
		return
	}

	edrDir := "./edr_files"
	if err := os.MkdirAll(edrDir, 0755); err != nil {
		log.Printf("创建edr_files文件夹失败: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	timestamp := time.Now().UnixNano()
	safeFilename := fmt.Sprintf("%d-%s", timestamp, filename)
	filePath := filepath.Join(edrDir, safeFilename)

	if err := ioutil.WriteFile(filePath, content, 0644); err != nil {
		log.Printf("保存可疑文件失败: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	clientIP := getClientIP(r)
	log.Printf("从 %s 获得了可疑文件: %s 路径: %s (%d bytes)",
		clientIP, suspFile.Filename, suspFile.Path, len(content))

	w.WriteHeader(http.StatusOK)
}
