package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
)

type WebhookData struct {
	Event string `json:"event"`
}

func testSendToWebhook(writer http.ResponseWriter, request *http.Request) {
	var requestData struct {
		URL     string `json:"url"`
		Content string `json:"content"`
	}

	if err := json.NewDecoder(request.Body).Decode(&requestData); err != nil {
		http.Error(writer, "Failed to parse JSON", http.StatusBadRequest)
		return
	}

	url := requestData.URL
	content := requestData.Content
	log.Println(url + " " + content)

	if url == "" || content == "" {
		http.Error(writer, "Missing Input", http.StatusBadRequest)
		return
	}

	sendToWebhook(url, content)

	writer.Header().Set("Content-Type", "application/json")

	json.NewEncoder(writer).Encode(map[string]string{"message": "Sent to webhook"})
}

func sendToWebhook(url string, content string) {
	data := WebhookData{
		Event: content,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("unexpected error %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("unexpected error %v", err)
		return
	}
	defer resp.Body.Close()
}
