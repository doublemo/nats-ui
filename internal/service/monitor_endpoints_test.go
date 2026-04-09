package service

import (
	"reflect"
	"testing"

	"github.com/doublemo/nats-ui/internal/models"
)

func TestDeriveMonitorEndpoints(t *testing.T) {
	got := deriveMonitorEndpoints([]string{
		"nats://127.0.0.1:4222",
		"127.0.0.1:4222",
		"tls://nats.example.com:4333",
	})

	want := []string{
		"http://127.0.0.1:8222",
		"http://nats.example.com:8333",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("deriveMonitorEndpoints() = %v, want %v", got, want)
	}
}

func TestEffectiveMonitorEndpointsPrefersConfigured(t *testing.T) {
	config := models.ConnectionConfig{
		NATSURLs:         []string{"nats://127.0.0.1:4222"},
		MonitorEndpoints: []string{"http://192.168.1.10:8222"},
	}

	got := effectiveMonitorEndpoints(config, "nats://10.0.0.8:4222")
	want := []string{"http://192.168.1.10:8222"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("effectiveMonitorEndpoints() = %v, want %v", got, want)
	}
}

func TestEffectiveMonitorEndpointsFallsBackToConnectedURLAndNATSURLs(t *testing.T) {
	config := models.ConnectionConfig{
		NATSURLs: []string{"nats://127.0.0.1:4222"},
	}

	got := effectiveMonitorEndpoints(config, "nats://10.0.0.8:4222")
	want := []string{
		"http://10.0.0.8:8222",
		"http://127.0.0.1:8222",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("effectiveMonitorEndpoints() = %v, want %v", got, want)
	}
}
