package logistics

import (
	"testing"
	"time"
)

func TestClassify(t *testing.T) {
	now := time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		input     ShipmentEvent
		wantEvent string
		wantErr   bool
	}{
		{
			name:      "device remains in transit",
			input:     ShipmentEvent{ShipmentID: "SHP-42", DeviceID: "DEV-7", Status: "moving", TemperatureC: 4.5, OccurredAt: now},
			wantEvent: "shipment.in_transit",
		},
		{
			name:      "temperature excursion becomes exception",
			input:     ShipmentEvent{ShipmentID: "SHP-42", DeviceID: "DEV-7", Status: "moving", TemperatureC: 11, OccurredAt: now},
			wantEvent: "shipment.exception",
		},
		{
			name: "signed delivery is complete",
			input: ShipmentEvent{ShipmentID: "SHP-42", DeviceID: "DEV-7", Status: "delivered", TemperatureC: 5, OccurredAt: now,
				ProofOfDelivery: &ProofOfDelivery{ObjectKey: "pod/SHP-42.pdf", Signer: "A. Chen"}},
			wantEvent: "shipment.delivered",
		},
		{
			name:    "delivery without proof is rejected",
			input:   ShipmentEvent{ShipmentID: "SHP-42", DeviceID: "DEV-7", Status: "delivered", TemperatureC: 5, OccurredAt: now},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Classify(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Classify() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got.Event != tt.wantEvent {
				t.Fatalf("Classify() event = %q, want %q", got.Event, tt.wantEvent)
			}
		})
	}
}
