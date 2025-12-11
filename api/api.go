package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"czechia.dev/probes"
	echoSwagger "github.com/swaggo/http-swagger"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

const alive = true

func isAlive() error {
	if alive {
		return nil
	}
	return errors.New("application is not alive")
}

func ProcessREST() {
	// Start liveness and readiness probes
	go probes.StartProbes(isAlive)

	// Create HTTP handler
	r := NewHandler()

	allowedOrigins := handlers.AllowedOrigins([]string{"*"})
	allowedMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	allowedHeaders := handlers.AllowedHeaders([]string{"authorization", "content-type", "accept", "origin", "cache-control", "x-requested-with", "ulang", "accept-language"})

	corsHeader := handlers.CORS(allowedOrigins, allowedMethods, allowedHeaders)

	server := &http.Server{
		Addr:         ":8901",
		Handler:      corsHeader(r),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start HTTP server
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Listen error :: ", err)
	}
}
func NewHandler() http.Handler {
	r := mux.NewRouter()

	//copy
	r.HandleFunc("/v1/migration/sync/copy", copy).Methods("POST")
	//sync
	r.HandleFunc("/v1/migration/sync/sync", sync).Methods("POST")
	//bucket list
	r.HandleFunc("/v1/migration/operations/list", bucketList).Methods("POST")

	// Register probes endpoints
	r.HandleFunc("/actuator/health/liveness", probeRoute(probes.Liveness)).Methods("GET")
	r.HandleFunc("/actuator/health/readiness", probeRoute(probes.Readiness)).Methods("GET")

	r.PathPrefix("/swagger").Handler(echoSwagger.WrapHandler).Methods("GET")

	return r
}

func probeRoute(p *probes.Probe) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if p.IsUp() {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}
}
