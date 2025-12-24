package infra

import (
	"sync"
)

type Registry struct {
	mu     sync.RWMutex
	config Config
	docker *DockerClient
	ollama *OllamaClient
	gpu    GPUProvider
}

var (
	registryOnce sync.Once
	registry     *Registry
)

func GetRegistry() *Registry {
	registryOnce.Do(func() {
		registry = &Registry{
			config: DefaultConfig(),
		}
	})
	return registry
}

func GetRegistryWithConfig(config Config) *Registry {
	registryOnce.Do(func() {
		registry = &Registry{
			config: config,
		}
	})
	return registry
}

func (r *Registry) Config() Config {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config
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

func (r *Registry) Ollama() (*OllamaClient, error) {
	r.mu.RLock()
	if r.ollama != nil {
		r.mu.RUnlock()
		return r.ollama, nil
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ollama != nil {
		return r.ollama, nil
	}

	docker, err := r.dockerUnsafe()
	if err != nil {
		return nil, err
	}

	r.ollama = NewOllamaClient(docker, r.config.OllamaBaseURL)
	return r.ollama, nil
}

func (r *Registry) dockerUnsafe() (*DockerClient, error) {
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

func (r *Registry) GPU() GPUProvider {
	r.mu.RLock()
	if r.gpu != nil {
		r.mu.RUnlock()
		return r.gpu
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.gpu != nil {
		return r.gpu
	}

	r.gpu = DetectGPU()
	return r.gpu
}

func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.docker != nil {
		r.docker.Close()
		r.docker = nil
	}
	r.ollama = nil
	r.gpu = nil
	return nil
}

func ResetRegistry() {
	if registry != nil {
		registry.Close()
	}
	registryOnce = sync.Once{}
	registry = nil
}

func GetSharedDockerClient() (*DockerClient, error) {
	return GetRegistry().Docker()
}

func ResetSharedDockerClient() {
	ResetRegistry()
}
