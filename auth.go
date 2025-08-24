package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type AuthAttempt struct {
	Count     int
	LastTry   time.Time
	BlockedAt *time.Time
}

type AuthManager struct {
	attempts map[string]*AuthAttempt
	mutex    sync.RWMutex
}

var authManager = &AuthManager{
	attempts: make(map[string]*AuthAttempt),
}

const (
	MaxAuthAttempts = 3
	BlockDuration   = 15 * time.Minute
)

func getUIAuthToken() string {
	token := os.Getenv("UI_AUTH_TOKEN")
	if token == "" {
		log.Printf("Warning: UI_AUTH_TOKEN not set")
		return "default_token"
	}
	return token
}

func (am *AuthManager) isBlocked(ip string) bool {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	
	attempt, exists := am.attempts[ip]
	if !exists {
		return false
	}
	
	if attempt.BlockedAt != nil {
		if time.Since(*attempt.BlockedAt) < BlockDuration {
			return true
		}
		// Unblock after duration
		attempt.BlockedAt = nil
		attempt.Count = 0
	}
	
	return false
}

func (am *AuthManager) recordAttempt(ip string, success bool) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	
	attempt, exists := am.attempts[ip]
	if !exists {
		attempt = &AuthAttempt{}
		am.attempts[ip] = attempt
	}
	
	if success {
		// Reset on successful login
		attempt.Count = 0
		attempt.BlockedAt = nil
		return
	}
	
	attempt.Count++
	attempt.LastTry = time.Now()
	
	if attempt.Count >= MaxAuthAttempts {
		now := time.Now()
		attempt.BlockedAt = &now
		log.Printf("IP %s blocked after %d failed attempts", ip, attempt.Count)
	}
}

func getClientIP(r *http.Request) string {
	// Check for forwarded IP first (for reverse proxies)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	return r.RemoteAddr
}

func authHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	clientIP := getClientIP(r)
	
	// Check if IP is blocked
	if authManager.isBlocked(clientIP) {
		log.Printf("Blocked IP %s attempted login", clientIP)
		http.Error(w, "Too many failed attempts. Try again later.", http.StatusTooManyRequests)
		return
	}
	
	var authData struct {
		Token string `json:"token"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&authData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		authManager.recordAttempt(clientIP, false)
		return
	}
	
	expectedToken := getUIAuthToken()
	success := authData.Token == expectedToken
	
	authManager.recordAttempt(clientIP, success)
	
	if success {
		// Create a simple session token (in production, use proper JWT or sessions)
		sessionToken := generateSessionToken()
		
		http.SetCookie(w, &http.Cookie{
			Name:     "auth_session",
			Value:    sessionToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   false, // Set to true in production with HTTPS
			MaxAge:   3600,  // 1 hour
		})
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	} else {
		log.Printf("Failed auth attempt from %s", clientIP)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid token"})
	}
}

func generateSessionToken() string {
	// Simple session token - in production use proper random generation
	return "session_" + time.Now().Format("20060102150405")
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("auth_session")
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		
		// In production, validate the session token properly
		if cookie.Value[:8] != "session_" {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		
		next(w, r)
	}
}

func loginPageHandler(w http.ResponseWriter, r *http.Request) {
	// Check if already authenticated
	if cookie, err := r.Cookie("auth_session"); err == nil && cookie.Value != "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	
	http.ServeFile(w, r, "static/login.html")
}