package main

import (
	"encoding/json"
	"html/template"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

func renderTemplate(w http.ResponseWriter, r *http.Request, templateName string, data PageData, templates *template.Template, podName string, namespace string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	
	if err := templates.ExecuteTemplate(w, templateName, data); err != nil {
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
}



func main() {
	// Initialize MIME types
	mime.AddExtensionType(".css", "text/css; charset=utf-8")
	mime.AddExtensionType(".js", "application/javascript; charset=utf-8")
	
	// Read environment variables with defaults
	title := getEnv("APP_TITLE", "Welcome to My Landing Page!")
	description := getEnv("APP_DESCRIPTION", "A simple, configurable landing page built with Go")
	version := getEnv("APP_VERSION", "1.0.0")
	environment := getEnv("APP_ENV", "development")
	contact := getEnv("APP_CONTACT", "contact@example.com")
	
	// Get Kubernetes pod metadata
	podName := getEnv("POD_NAME", "local-dev")
	namespace := getEnv("POD_NAMESPACE", "default")

	// Parse HTML templates
	templates, err := template.ParseFiles(
		"templates/index.html",
		"templates/about.html",
		"templates/docs.html",
		"templates/pricing.html",
		"templates/contact.html",
		"templates/pokemon.html",
	)
	if err != nil {
		log.Fatal("Error parsing templates:", err)
	}

	// Setup HTTP routes
	mux := http.NewServeMux()
	
	// Common page data
	basePageData := PageData{
		Title:       title,
		Description: description,
		Version:     version,
		Environment: environment,
		Contact:     contact,
	}

	// Home page handler
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		pageData := basePageData
		pageData.Title = title
		renderTemplate(w, r, "index.html", pageData, templates, podName, namespace)
	})

	// About page handler
	mux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		pageData := basePageData
		pageData.Title = "About - " + title
		renderTemplate(w, r, "about.html", pageData, templates, podName, namespace)
	})

	// Documentation page handler
	mux.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		pageData := basePageData
		pageData.Title = "Documentation - " + title
		renderTemplate(w, r, "docs.html", pageData, templates, podName, namespace)
	})

	// Pricing page handler
	mux.HandleFunc("/pricing", func(w http.ResponseWriter, r *http.Request) {
		pageData := basePageData
		pageData.Title = "Pricing - " + title
		renderTemplate(w, r, "pricing.html", pageData, templates, podName, namespace)
	})

	// Contact page handler
	mux.HandleFunc("/contact", func(w http.ResponseWriter, r *http.Request) {
		pageData := basePageData
		pageData.Title = "Contact - " + title
		renderTemplate(w, r, "contact.html", pageData, templates, podName, namespace)
	})

	// Pokémon page handler
	mux.HandleFunc("/pokemon", func(w http.ResponseWriter, r *http.Request) {
		pageData := basePageData
		pageData.Title = "Pokémon Database - " + title
		pageData.Description = "Explore Pokémon data, types, and abilities"
		renderTemplate(w, r, "pokemon.html", pageData, templates, podName, namespace)
	})

	// Static files handler (CSS, JS, images) with proper MIME types
	mux.Handle("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the file path
		path := r.URL.Path[1:] // Remove leading slash
		
		// Determine MIME type
		ext := strings.ToLower(filepath.Ext(path))
		contentType := mime.TypeByExtension(ext)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		
		// Force correct MIME types for common static files
		switch ext {
		case ".css":
			contentType = "text/css; charset=utf-8"
		case ".js":
			contentType = "application/javascript; charset=utf-8"
		case ".jpg", ".jpeg":
			contentType = "image/jpeg"
		case ".png":
			contentType = "image/png"
		case ".gif":
			contentType = "image/gif"
		case ".svg":
			contentType = "image/svg+xml"
		case ".ico":
			contentType = "image/x-icon"
		case ".woff":
			contentType = "font/woff"
		case ".woff2":
			contentType = "font/woff2"
		}
		
		// Set headers
		w.Header().Set("Content-Type", contentType)
		// Add caching headers for static assets
		w.Header().Set("Cache-Control", "public, max-age=31536000") // 1 year
		
		// Serve file
		http.ServeFile(w, r, path)
	}))
	
	// Wrap handler with access logger
	logger := &accessLogger{
		handler:   mux,
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
