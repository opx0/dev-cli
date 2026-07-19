package infra

import "sync"

var (
	dockerOnce sync.Once
	docker     *DockerClient
	dockerErr  error
)

func GetSharedDockerClient() (*DockerClient, error) {
	dockerOnce.Do(func() { docker, dockerErr = NewDockerClient() })
	return docker, dockerErr
}
