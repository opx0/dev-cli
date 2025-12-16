package infra

import (
	"fmt"
	"net"
	"time"
)

type ServiceStatus struct {
	Name      string
	Port      int
	Available bool
	Error     error
}

func CheckServices() []ServiceStatus {
	services := []struct {
		name string
		port int
	}{
		{"Postgres", 5432},
		{"Redis", 6379},
		{"Ollama", 11434},
	}

	var results []ServiceStatus

	for _, s := range services {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", s.port), 500*time.Millisecond)
		status := ServiceStatus{
			Name: s.name,
			Port: s.port,
		}

		if err != nil {
			status.Available = false
			status.Error = err
		} else {
			status.Available = true
			conn.Close()
		}
		results = append(results, status)
	}

	return results
}
