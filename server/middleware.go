package main

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
)

func basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			w.Header().Set("WWW-Authenticate", `Basic realm=""`)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
			return
		}

		if !strings.HasPrefix(auth, "Basic ") {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid authentication method"))
			return
		}

		payload, err := base64.StdEncoding.DecodeString(auth[6:])
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid base64 encoding"))
			return
		}

		pair := strings.SplitN(string(payload), ":", 2)
		if len(pair) != 2 {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid credentials format"))
			return
		}

		user, pass := pair[0], pair[1]
		usernameMatch := subtle.ConstantTimeCompare([]byte(user), []byte(*username)) == 1
		passwordMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(*password)) == 1

		if !usernameMatch || !passwordMatch {
			w.Header().Set("WWW-Authenticate", `Basic realm=""`)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid credentials"))
			return
		}

		next(w, r)
	}
}
