package main

type Client struct {
	IP          string `json:"ip"`
	Hostname    string `json:"hostname"`
	UserInfo    string `json:"userinfo"`
	ProcessInfo string `json:"processinfo"`
	LastSeen    string `json:"last_seen"`
	RevShell    int    `json:"revshell"`
}

type Message struct {
	Timestamp   int64  `json:"timestamp"`
	MessageType string `json:"message_type"` // warning, info, success
	Title       string `json:"title"`
	Content     string `json:"content"`
}

type HeartbeatData struct {
	Hostname    string `json:"hostname"`
	UserInfo    string `json:"userinfo"`
	ProcessInfo string `json:"processinfo"`
}

type SuspiciousFile struct {
	Filename string `json:"filename"`
	Path     string `json:"path"`
	Content  string `json:"content"`
}

type FlagSubmission struct {
	Flag string `json:"flag"`
}
