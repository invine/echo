package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

func handleEchoGet(w http.ResponseWriter, r *http.Request) {
	message := r.URL.Query().Get("message")
	if message == "" {
		fmt.Fprintf(w, "Missing 'message' parameter.")
		slog.Warn("received request without message", "method", "GET", "remoteAddr", r.RemoteAddr)
		return
	}
	fmt.Fprintf(w, "Received message (GET): %s", message)
	slog.Debug("received message", "method", "GET", "message", message, "remoteAddr", r.RemoteAddr)
}

func handleEchoPost(w http.ResponseWriter, r *http.Request) {
	var requestMessage map[string]string
	defer r.Body.Close()
	b, _ := io.ReadAll(r.Body)
	// decoder := json.NewDecoder(r.Body)
	err := json.Unmarshal(b, &requestMessage) //decoder.Decode(&requestMessage)
	if err != nil {
		fmt.Fprintf(w, "body is formatted incorrectly")
		slog.Warn("can't parse JSON", "method", "POST", "remoteAddr", r.RemoteAddr, "body", string(b), "err", err)
		return
	}

	message, ok := requestMessage["message"]
	if !ok {
		fmt.Fprintf(w, "Missing 'message' parameter.")
		slog.Warn("received request without message", "method", "POST", "remoteAddr", r.RemoteAddr)
		return
	}

	fmt.Fprintf(w, "Received message (POST): %s", message)
	slog.Debug("received message", "method", "POST", "message", message, "remoteAddr", r.RemoteAddr)
}

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /echo", handleEchoGet)
	mux.HandleFunc("POST /echo", handleEchoPost)
	slog.Info("starting http server...")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		fmt.Println("Error serving:", err)
	}
}
