package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/invine/echo/internal/config"
)

func handleGetEcho(w http.ResponseWriter, r *http.Request) {
	message := r.URL.Query().Get("message")
	if message == "" {
		fmt.Fprintf(w, "Missing 'message' parameter.")
		slog.Warn("received request without message",
			"method", "GET",
			"url", r.URL.String(),
			"remoteAddr", r.RemoteAddr,
			"headers", r.Header,
		)
		return
	}
	fmt.Fprintf(w, "Received message (GET): %s", message)
	slog.Debug("received message",
		"method", "GET",
		"url", r.URL.String(),
		"message", message,
		"remoteAddr", r.RemoteAddr,
		"headers", r.Header,
	)
}

func handlePostEcho(w http.ResponseWriter, r *http.Request) {
	var requestBody map[string]string
	defer r.Body.Close()
	requestBodyBytes, _ := io.ReadAll(r.Body)
	err := json.Unmarshal(requestBodyBytes, &requestBody)
	if err != nil {
		fmt.Fprintf(w, "body is formatted incorrectly")
		slog.Warn("can't parse JSON",
			"method", "POST",
			"url", r.URL.String(),
			"remoteAddr", r.RemoteAddr,
			"body", string(requestBodyBytes),
			"err", err,
			"headers", r.Header,
		)
		return
	}

	message, ok := requestBody["message"]
	if !ok {
		fmt.Fprintf(w, "Missing 'message' parameter.")
		slog.Warn("received request without message",
			"method", "POST",
			"url", r.URL.String(),
			"remoteAddr", r.RemoteAddr,
			"body", string(requestBodyBytes),
			"headers", r.Header,
		)
		return
	}

	fmt.Fprintf(w, "Received message (POST): %s", message)
	slog.Debug("received message",
		"method", "POST",
		"url", r.URL.String(),
		"message", message,
		"remoteAddr", r.RemoteAddr,
		"body", string(requestBodyBytes),
		"headers", r.Header,
	)
}

func handleGetPing(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "pong")
	slog.Debug("received ping",
		"method", "GET",
		"url", r.URL.String(),
		"remoteAddr", r.RemoteAddr,
		"headers", r.Header,
	)
}

func main() {
	config, err := config.NewConfig()
	if err != nil {
		slog.Error(fmt.Sprintf("initializing config: %v", err))
		return
	}

	setLogLevel(config)

	// Initialize the HTTP server
	httpServerMux := http.NewServeMux()
	httpServerMux.HandleFunc("GET /", handleGetEcho)
	httpServerMux.HandleFunc("POST /", handlePostEcho)
	httpServerMux.HandleFunc("GET /ping", handleGetPing)
	slog.Info("starting http server...")
	if err := http.ListenAndServe(":8080", httpServerMux); err != nil {
		fmt.Println("Error serving:", err)
	}
}

func setLogLevel(config *config.Config) {
	// Set the log level.
	switch config.LogLevel {
	case "debug":
		slog.SetLogLoggerLevel(slog.LevelDebug)
	case "info":
		slog.SetLogLoggerLevel(slog.LevelInfo)
	case "warn":
		slog.SetLogLoggerLevel(slog.LevelWarn)
	case "error":
		slog.SetLogLoggerLevel(slog.LevelError)
	default:
		slog.SetLogLoggerLevel(slog.LevelInfo)
		slog.Warn("invalid log level, defaulting to info", "logLevel", config.LogLevel)
		return
	}
	slog.Info("setting log level", "logLevel", config.LogLevel)
}
