package main

import (
	"database/sql"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func initDB() error {
	var err error
	db, err = sql.Open("sqlite", "./awd-server.db?_busy_timeout=10000&_journal_mode=WAL&_synchronous=NORMAL&_cache_size=1000")
	if err != nil {
		return err
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	schemas := []string{
		`CREATE TABLE IF NOT EXISTS clients (
			ip TEXT PRIMARY KEY,
			hostname TEXT,
			userinfo TEXT,
			processinfo TEXT,
			last_seen TEXT,
			revshell INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			timestamp INTEGER PRIMARY KEY,
			message_type TEXT,
			title TEXT,
			content TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS config (
			key TEXT PRIMARY KEY,
			value TEXT
		)`,
	}

	for _, schema := range schemas {
		if _, err := db.Exec(schema); err != nil {
			return err
		}
	}

	defaultConfigs := map[string]string{
		"revshell_addr":   "192.168.1.1:1337",
		"submit_template": "curl http://192.168.1.1/submit -X POST -H 'Content-Type: application/json' -d '{\"flag\": \"{FLAG}\"}'",
	}

	for key, value := range defaultConfigs {
		_, err := db.Exec("INSERT OR IGNORE INTO config (key, value) VALUES (?, ?)", key, value)
		if err != nil {
			return err
		}
	}

	return nil
}

func addMessage(msgType, title, content string) {
	timestamp := time.Now().UnixNano()
	_, err := db.Exec("INSERT INTO messages (timestamp, message_type, title, content) VALUES (?, ?, ?, ?)",
		timestamp, msgType, title, content)
	if err != nil {
		log.Printf("增加消息失败: %v", err)
	}
}
