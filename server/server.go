package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Sah-Abhishek/mockSmith/config"
)

type Server struct {
	config     *config.Config
	mux        *http.ServeMux
	mu         sync.RWMutex
	updateChan <-chan *config.Config
}

func Start(cfg *config.Config, updateChan <-chan *config.Config) error {
	s := &Server{
		config:     cfg,
		mux:        http.NewServeMux(),
		updateChan: updateChan,
	}

	// Initial route setup
	s.setupRoutes()

	// Listen for config updates
	go s.watchForUpdates()

	addr := fmt.Sprintf(":%d", cfg.Port)
	fmt.Printf("🚀 Mock API Server running on http://localhost%s\n", addr)
	fmt.Println("Press 'a' in TUI to add endpoints")
	fmt.Println()

	return http.ListenAndServe(addr, s)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Enable CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	s.mux.ServeHTTP(w, r)
}

func (s *Server) setupRoutes() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create new mux
	s.mux = http.NewServeMux()

	// Add all endpoints
	for _, endpoint := range s.config.GetEndpoints() {
		s.addRoute(endpoint)
	}

	// Default 404 handler
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Endpoint not found. Add it via the TUI!",
		})
	})
}

func (s *Server) addRoute(endpoint config.Endpoint) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Check method
		if r.Method != endpoint.Method {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Simulate delay if specified
		if endpoint.Delay > 0 {
			time.Sleep(time.Duration(endpoint.Delay) * time.Millisecond)
		}

		// Set custom headers
		w.Header().Set("Content-Type", "application/json")
		for k, v := range endpoint.Headers {
			w.Header().Set(k, v)
		}

		// Log request
		fmt.Printf("[%s] %s %s -> %d\n",
			time.Now().Format("15:04:05"),
			endpoint.Method,
			endpoint.Path,
			endpoint.StatusCode,
		)

		// Send response
		w.WriteHeader(endpoint.StatusCode)
		w.Write(endpoint.Response)
	}

	s.mux.HandleFunc(endpoint.Path, handler)
}

func (s *Server) watchForUpdates() {
	for cfg := range s.updateChan {
		s.config = cfg
		s.setupRoutes()
		fmt.Printf("🔄 Routes reloaded (%d endpoints)\n", len(cfg.GetEndpoints()))
	}
}
