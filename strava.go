package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type StravaConfig struct {
	ClientID     string
	ClientSecret string
}

type WebhookSubscription struct {
	ID int `json:"id"`
}

type WebhookEvent struct {
	ObjectType     string                 `json:"object_type"`
	ObjectID       int64                  `json:"object_id"`
	AspectType     string                 `json:"aspect_type"`
	Updates        map[string]interface{} `json:"updates"`
	OwnerID        int64                  `json:"owner_id"`
	SubscriptionID int                    `json:"subscription_id"`
	EventTime      int64                  `json:"event_time"`
}

func getStravaConfig() StravaConfig {
	// TODO:  I should move this to config folder
	clientID := os.Getenv("STRAVA_CLIENT_ID")
	if clientID == "" {
		log.Printf("Warning: STRAVA_CLIENT_ID not set")
	}

	clientSecret := os.Getenv("STRAVA_CLIENT_SECRET")
	if clientSecret == "" {
		log.Printf("Warning: STRAVA_CLIENT_SECRET not set")
	}

	return StravaConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}
}

func getUrlsToFowardTo() []string {
	return strings.Split(os.Getenv("FORWARD_URLS"), ",")
}

func getWebhookCallbackURL() string {
	if baseURL := os.Getenv("WEBHOOK_BASE_URL"); baseURL != "" {
		return baseURL + "/webhook"
	}

	// To avoid any issues with establishing connection to local host
	return "BAD_URL"
}

func getWebhookVerifyToken() string {
	if token := os.Getenv("STRAVA_WEBHOOK_VERIFY_TOKEN"); token != "" {
		return token
	}
	return "STRAVA_WEBHOOK_VERIFY_TOKEN"
}

func stravaAuthURLHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClientID    string `json:"client_id"`
		RedirectURI string `json:"redirect_uri"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	authURL := fmt.Sprintf("https://www.strava.com/oauth/authorize?client_id=%s&response_type=code&redirect_uri=%s&approval_prompt=force&scope=read,activity:read",
		req.ClientID, url.QueryEscape(req.RedirectURI))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"auth_url": authURL})
}

func stravaWebhookGetHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received GET request from %s", r.RemoteAddr)
	handleWebhookVerification(w, r)
}

func stravaWebhookPostHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received POST request from %s", r.RemoteAddr)
	log.Printf("URL: %s", r.URL.String())

	handleWebhookEvent(w, r)
}

func handleWebhookVerification(w http.ResponseWriter, r *http.Request) {
	log.Printf("Verification request received from %s", r.RemoteAddr)

	challenge := r.URL.Query().Get("hub.challenge")
	verifyToken := r.URL.Query().Get("hub.verify_token")
	mode := r.URL.Query().Get("hub.mode")

	expectedToken := getWebhookVerifyToken()

	if mode != "subscribe" {
		log.Printf("Invalid mode: expected 'subscribe', got '%s'", mode)
		http.Error(w, "Invalid mode", http.StatusBadRequest)
		return
	}

	if verifyToken != expectedToken {
		log.Printf("Token mismatch")
		http.Error(w, "Invalid verify token", http.StatusForbidden)
		return
	}

	if challenge == "" {
		log.Printf("Missing challenge parameter")
		http.Error(w, "Missing challenge", http.StatusBadRequest)
		return
	}

	log.Printf(" Verification successful, responding with challenge")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]string{
		"hub.challenge": challenge,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return
	}

	log.Printf("Sending response")
	_, writeErr := w.Write(responseBytes)
	if writeErr != nil {
		log.Printf("Failed to write response: %v", writeErr)
	} else {
		log.Printf("Successfully sent verification response")
	}
}

func handleWebhookEvent(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	log.Printf("Raw request body: %s", string(bodyBytes))

	var event WebhookEvent
	if err := json.Unmarshal(bodyBytes, &event); err != nil {
		log.Printf("Failed to parse webhook event JSON: %v", err)
		log.Printf("Invalid JSON body: %s", string(bodyBytes))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("Parsed event - ObjectType: %s, ObjectID: %d, AspectType: %s, OwnerID: %d, SubscriptionID: %d, EventTime: %d",
		event.ObjectType, event.ObjectID, event.AspectType, event.OwnerID, event.SubscriptionID, event.EventTime)

	if len(event.Updates) > 0 {
		log.Printf("Event updates: %v", event.Updates)
	}

	for _, url := range getUrlsToFowardTo() {
		log.Printf("Forwarding Event to %s", url)
		sendToWebhook(url, string(bodyBytes))
	}

	w.WriteHeader(http.StatusOK)
}

func createWebhookSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	config := getStravaConfig()
	callbackURL := getWebhookCallbackURL()
	verifyToken := getWebhookVerifyToken()

	log.Printf("Creating subscription with callback URL: %s", callbackURL)

	subscriptionURL := "https://www.strava.com/api/v3/push_subscriptions"
	requestBody := map[string]string{
		"client_id":     config.ClientID,
		"client_secret": config.ClientSecret,
		"callback_url":  callbackURL,
		"verify_token":  verifyToken,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		http.Error(w, "Failed to marshal request", http.StatusInternalServerError)
		return
	}

	resp, err := http.Post(subscriptionURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		http.Error(w, "Failed to create subscription", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		errorBody := string(bodyBytes)
		log.Printf("Strava API error: %d - %s", resp.StatusCode, errorBody)
		http.Error(w, fmt.Sprintf("Strava API returned status %d: %s", resp.StatusCode, errorBody), http.StatusBadRequest)
		return
	}

	var subscription WebhookSubscription
	if err := json.NewDecoder(resp.Body).Decode(&subscription); err != nil {
		http.Error(w, "Failed to parse subscription response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscription)
}
