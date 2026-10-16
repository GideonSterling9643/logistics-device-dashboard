package logistics

import (
	"errors"
	"strings"
	"time"
)

type ProofOfDelivery struct {
	ObjectKey string `json:"object_key"`
	Signer    string `json:"signer"`
}

type ShipmentEvent struct {
	ShipmentID      string           `json:"shipment_id"`
	DeviceID        string           `json:"device_id"`
	Status          string           `json:"status"`
	TemperatureC    float64          `json:"temperature_c"`
	Damaged         bool             `json:"damaged"`
	Exception       string           `json:"exception,omitempty"`
	OccurredAt      time.Time        `json:"occurred_at"`
	ProofOfDelivery *ProofOfDelivery `json:"proof_of_delivery,omitempty"`
}

type DashboardUpdate struct {
	Event string        `json:"event"`
	Data  ShipmentEvent `json:"data"`
}

func Classify(event ShipmentEvent) (DashboardUpdate, error) {
	if strings.TrimSpace(event.ShipmentID) == "" || strings.TrimSpace(event.DeviceID) == "" {
		return DashboardUpdate{}, errors.New("shipment_id and device_id are required")
	}
	if event.OccurredAt.IsZero() {
		return DashboardUpdate{}, errors.New("occurred_at is required")
	}

	kind := "shipment.in_transit"
	if event.Damaged || event.Exception != "" || event.TemperatureC < 2 || event.TemperatureC > 8 {
		kind = "shipment.exception"
	} else if event.Status == "delivered" {
		if event.ProofOfDelivery == nil || strings.TrimSpace(event.ProofOfDelivery.ObjectKey) == "" {
			return DashboardUpdate{}, errors.New("delivered events require proof_of_delivery.object_key")
		}
		kind = "shipment.delivered"
	}

	return DashboardUpdate{Event: kind, Data: event}, nil
}
