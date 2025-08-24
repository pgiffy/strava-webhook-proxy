package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return ":" + port
}
func main() {
	router := mux.NewRouter()
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))
	
	// Authentication routes (no auth required)
	router.HandleFunc("/login", loginPageHandler).Methods("GET")
	router.HandleFunc("/auth", authHandler).Methods("POST")
	
	// Protected routes (auth required)
	router.HandleFunc("/", authMiddleware(homeHandler)).Methods("GET")
	router.HandleFunc("/sendToWebhook", authMiddleware(manuallySendToWebhook)).Methods("POST")
	router.HandleFunc("/create-subscription", authMiddleware(createWebhookSubscription)).Methods("POST")
	router.HandleFunc("/auth/callback", authMiddleware(stravaAuthCallbackHandler)).Methods("GET")

	// Strava webhook endpoints (no auth required - these are called by Strava)
	router.HandleFunc("/webhook", stravaWebhookGetHandler).Methods("GET")
	router.HandleFunc("/webhook", stravaWebhookPostHandler).Methods("POST")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	addr := getPort()
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		fmt.Printf("Server starting on %s\n", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start:", err)
		}
	}()

	<-c
	fmt.Println("\nShutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/index.html")
}
