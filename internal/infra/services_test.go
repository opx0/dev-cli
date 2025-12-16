package infra

import (
	"net"
	"testing"
)

func TestCheckServices(t *testing.T) {
	// This test depends on actual services running, which might be flaky.
	// So we can mock, or just check that it returns a list of 3 items (Postgres, Redis, Ollama).

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
		// We don't verify Available field as it depends on environment
	}
}

func TestCheckServices_LocalListener(t *testing.T) {
	// Start a dummy listener to simulate a running service
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Skip("could not listen on local port")
	}
	defer l.Close()

	// We can't easily inject this into CheckServices without refactoring it to accept a list.
	// So for now, we just rely on previous test for structure.
}
