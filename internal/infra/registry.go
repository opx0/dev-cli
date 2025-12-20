package infra

import (
	"sync"
)

type Registry struct {
	mu     sync.RWMutex
	docker *DockerClient
}

var (
	registryOnce sync.Once
	registry     *Registry
)

func GetRegistry() *Registry {
	registryOnce.Do(func() {
		registry = &Registry{}
	})
	return registry
}

func (r *Registry) Docker() (*DockerClient, error) {
	r.mu.RLock()
	if r.docker != nil {
		r.mu.RUnlock()
		return r.docker, nil
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.docker != nil {
		return r.docker, nil
	}

	client, err := NewDockerClient()
	if err != nil {
		return nil, err
	}
	r.docker = client
	return r.docker, nil
}

func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.docker != nil {
		r.docker.Close()
		r.docker = nil
	}
	return nil
}

func ResetRegistry() {
	if registry != nil {
		registry.Close()
	}
	registryOnce = sync.Once{}
	registry = nil
}
