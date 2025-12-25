package infra

import (
	"net"
	"testing"
)

func TestCheckServices(t *testing.T) {

	results := CheckServices()

	if len(results) != 3 {
		t.Errorf("expected 3 services, got %d", len(results))
	}

	expected := map[string]int{
		"Postgres": 5432,
		"Redis":    6379,
		"Ollama":   11434,
	}

	for _, res := range results {
		port, ok := expected[res.Name]
		if !ok {
			t.Errorf("unexpected service: %s", res.Name)
		}
		if res.Port != port {
			t.Errorf("expected port %d for %s, got %d", port, res.Name, res.Port)
		}

	}
}

func TestCheckServices_LocalListener(t *testing.T) {

	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Skip("could not listen on local port")
	}
	defer l.Close()

}
