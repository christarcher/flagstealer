package main

import (
	"net"
	"net/http"
	"strings"
)

func getClientIP(r *http.Request) string {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return strings.TrimSpace(ip)
}
