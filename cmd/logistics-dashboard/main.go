package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"

	gateway "example.com/logistics-device-dashboard/internal/infrai"
	"example.com/logistics-device-dashboard/internal/logistics"
)

type server struct {
	client  *gateway.Client
	channel string
	account string
}

func main() {
	setup := flag.Bool("setup", false, "create the configured realtime channel")
	flag.Parse()
	client, err := gateway.New(os.Getenv("INFRAI_BASE_URL"), os.Getenv("INFRAI_API_KEY"), nil)
	if err != nil {
		log.Fatal(err)
	}
	s := &server{client: client, channel: envOr("INFRAI_CHANNEL", "logistics-live"), account: envOr("INFRAI_ACCOUNT_ID", "logistics-ops")}
	if *setup {
		if err := client.CreateChannel(context.Background(), s.channel); err != nil {
			log.Fatal(err)
		}
		log.Printf("created realtime channel %s", s.channel)
		return
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /shipments/events", s.handleEvent)
	mux.HandleFunc("GET /dashboard/metrics", s.handleMetrics)
	addr := envOr("ADDR", ":8080")
	log.Printf("logistics dashboard listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func (s *server) handleEvent(w http.ResponseWriter, r *http.Request) {
	var event logistics.ShipmentEvent
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&event); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid shipment event"})
		return
	}
	update, err := logistics.Classify(event)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	idempotencyKey := strings.Join([]string{event.ShipmentID, event.DeviceID, event.OccurredAt.UTC().Format("20060102T150405.000000000Z")}, ":")
	if err := s.client.Publish(r.Context(), s.channel, update.Event, s.account, idempotencyKey, update.Data); err != nil {
		status := http.StatusBadGateway
		var apiErr *gateway.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
			status = apiErr.StatusCode
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, update)
}

func (s *server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	data, err := s.client.QueryMetrics(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
