package main

import (
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

func main() {
	// Read environment variables with defaults
	title := getEnv("APP_TITLE", "Welcome to My Landing Page")
	description := getEnv("APP_DESCRIPTION", "A simple, configurable landing page built with Go")
	version := getEnv("APP_VERSION", "1.0.0")
	environment := getEnv("APP_ENV", "development")
	contact := getEnv("APP_CONTACT", "contact@example.com")

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

	// Setup HTTP server with logging middleware
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		
		// Log the access request
		log.Printf("ACCESS: %s - %s %s - User-Agent: %s - Remote: %s", 
			startTime.Format("2006-01-02 15:04:05"), 
			r.Method, 
			r.URL.Path,
			r.Header.Get("User-Agent"),
			r.RemoteAddr,
		)
		
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, pageData); err != nil {
			log.Printf("Error executing template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		
		// Log response time
		log.Printf("RESPONSE: %s %s - Duration: %v", 
			r.Method, 
			r.URL.Path,
			time.Since(startTime),
		)
	})

	port := getEnv("PORT", "8080")
	log.Printf("Server starting on port %s", port)
	log.Printf("Environment: %s", environment)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}