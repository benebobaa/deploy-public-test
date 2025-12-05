package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"
)

type PageData struct {
	Title       string
	Description string
	Version     string
	Environment string
	Contact     string
}

type LogEntry struct {
	Timestamp   string `json:"timestamp"`
	Level       string `json:"level"`
	Message     string `json:"message"`
	Method      string `json:"method,omitempty"`
	Path        string `json:"path,omitempty"`
	StatusCode  int    `json:"status_code,omitempty"`
	Duration    string `json:"duration,omitempty"`
	UserAgent   string `json:"user_agent,omitempty"`
	RemoteAddr  string `json:"remote_addr,omitempty"`
	PodName     string `json:"pod_name,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
}

type accessLogger struct {
	handler    http.Handler
	podName    string
	namespace  string
}

func (al *accessLogger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	
	// Create a response writer wrapper to capture status code
	wrapper := &responseWriter{ResponseWriter: w, statusCode: 200}
	
	// Process request
	al.handler.ServeHTTP(wrapper, r)
	
	// Log in structured format for Kubernetes
	entry := LogEntry{
		Timestamp:   startTime.Format(time.RFC3339),
		Level:       "INFO",
		Message:     "HTTP request processed",
		Method:      r.Method,
		Path:        r.URL.Path,
		StatusCode:  wrapper.statusCode,
		Duration:    time.Since(startTime).String(),
		UserAgent:   r.Header.Get("User-Agent"),
		RemoteAddr:  r.RemoteAddr,
		PodName:     al.podName,
		Namespace:   al.namespace,
	}
	
	if logData, err := json.Marshal(entry); err == nil {
		log.Printf("%s", string(logData))
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func main() {
	// Read environment variables with defaults
	title := getEnv("APP_TITLE", "Welcome to My Landing Page")
	description := getEnv("APP_DESCRIPTION", "A simple, configurable landing page built with Go")
	version := getEnv("APP_VERSION", "1.0.0")
	environment := getEnv("APP_ENV", "development")
	contact := getEnv("APP_CONTACT", "contact@example.com")
	
	// Get Kubernetes pod metadata
	podName := getEnv("POD_NAME", "local-dev")
	namespace := getEnv("POD_NAMESPACE", "default")

	// Parse HTML template
	tmpl := template.Must(template.ParseFiles("templates/index.html"))

	// Prepare page data
	pageData := PageData{
		Title:       title,
		Description: description,
		Version:     version,
		Environment: environment,
		Contact:     contact,
	}

	// Setup HTTP handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, pageData); err != nil {
			// Log error in structured format
			errorEntry := LogEntry{
				Timestamp: time.Now().Format(time.RFC3339),
				Level:     "ERROR",
				Message:   err.Error(),
				Path:      r.URL.Path,
				PodName:   podName,
				Namespace: namespace,
			}
			if logData, err := json.Marshal(errorEntry); err == nil {
				log.Printf("%s", string(logData))
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	})
	
	// Wrap handler with access logger
	logger := &accessLogger{
		handler:   handler,
		podName:   podName,
		namespace: namespace,
	}

	port := getEnv("PORT", "8080")
	
	// Log startup in structured format
	startupEntry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     "INFO",
		Message:   "Server starting",
		PodName:   podName,
		Namespace: namespace,
	}
	if logData, err := json.Marshal(startupEntry); err == nil {
		log.Printf("%s", string(logData))
	}
	
	log.Fatal(http.ListenAndServe(":"+port, logger))
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}