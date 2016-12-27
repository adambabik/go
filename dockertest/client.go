// Package dockertest is inspired by github.com/ory-am/dockertest@v3.
//
package dockertest

import (
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cenk/backoff"
	dc "github.com/fsouza/go-dockerclient"
)

type (
	// ContainerList is a list of `*dc.Container`.
	ContainerList []*dc.Container

	// Pool manages created docker artifacts.
	Pool struct {
		Client *dc.Client

		rw         sync.RWMutex
		Containers ContainerList
		Networks   []*dc.Network
	}

	// Env is a list of environment variables in format NAME=VALUE.
	Env []string
)

// Remove removes an item from `ContainerList`.
func (c ContainerList) Remove(search *dc.Container) ContainerList {
	filtered := make(ContainerList, 0, len(c))
	for _, container := range c {
		if container.ID != search.ID {
			filtered = append(filtered, container)
		}
	}

	return filtered
}

func defaultEndpoint() string {
	if os.Getenv("DOCKER_URL") != "" {
		return os.Getenv("DOCKER_URL")
	} else if runtime.GOOS == "windows" {
		return "http://localhost:2375"
	}

	return "unix:///var/run/docker.sock"
}

func parseImageName(image string) (string, string) {
	if idx := strings.Index(image, ":"); idx != -1 {
		return image[:idx], image[idx+1:]
	}

	return image, "latest"
}

// NewPool creates a new client.
func NewPool(endpoint string) (*Pool, error) {
	if endpoint == "" {
		endpoint = defaultEndpoint()
	}

	dc, err := dc.NewClient(endpoint)
	if err != nil {
		return nil, err
	}

	return &Pool{Client: dc}, nil
}

// PullImage pulls image from the Docker Hub.
func (p *Pool) PullImage(image string) error {
	imageName, tag := parseImageName(image)
	return p.Client.PullImage(dc.PullImageOptions{
		Repository: imageName,
		Tag:        tag,
	}, dc.AuthConfiguration{})
}

// RunContainer runs a container with a given image and env vars.
// It's a short version of `RunContainerWithOpts()`.
func (p *Pool) RunContainer(
	image string, env Env, pullImage bool,
) (*dc.Container, error) {
	_, err := p.Client.InspectImage(image)
	if err != nil {
		if !pullImage {
			return nil, err
		}

		err := p.PullImage(image)
		if err != nil {
			return nil, err
		}
	}

	return p.RunContainerWithOpts(dc.CreateContainerOptions{
		Config: &dc.Config{
			Image: image,
			Env:   env,
		},
		HostConfig: &dc.HostConfig{
			PublishAllPorts: true,
			AutoRemove:      false,
		},
	})
}

// RunContainerWithOpts runs a container based on given options.
func (p *Pool) RunContainerWithOpts(
	opts dc.CreateContainerOptions,
) (*dc.Container, error) {
	container, err := p.Client.CreateContainer(opts)
	if err != nil {
		return nil, err
	}

	err = p.Client.StartContainer(container.ID, nil)
	if err != nil {
		return nil, err
	}

	container, err = p.Client.InspectContainer(container.ID)
	if err != nil {
		return nil, err
	}

	p.rw.Lock()
	p.Containers = append(p.Containers, container)
	p.rw.Unlock()

	return container, nil
}

// RunMultipleContainers spawns multiple containers asynchronously.
func (p *Pool) RunMultipleContainers(
	opts []dc.CreateContainerOptions,
) (ContainerList, error) {
	var wg sync.WaitGroup
	var rw sync.RWMutex

	containers := make(ContainerList, 0, len(opts))
	errCh := make(chan error, len(opts))

	// Async containers setup.
	for _, options := range opts {
		wg.Add(1)
		go func(options dc.CreateContainerOptions) {
			defer wg.Done()
			if c, errRun := p.RunContainerWithOpts(options); errRun != nil {
				errCh <- errRun
			} else {
				rw.Lock()
				containers = append(containers, c)
				rw.Unlock()
			}
		}(options)
	}

	wg.Wait()
	close(errCh)
	if err := <-errCh; err != nil {
		return containers, err
	}

	return containers, nil
}

// PurgeContainers removes containers passed as in the argument.
func (p *Pool) PurgeContainers(containers ContainerList) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(containers))

	for _, c := range containers {
		wg.Add(1)
		go func(container *dc.Container) {
			defer wg.Done()
			err := p.PurgeContainer(container)
			if err != nil {
				errCh <- err
			}
		}(c)
	}

	wg.Wait()
	close(errCh)
	if err := <-errCh; err != nil {
		return err
	}

	return nil
}

// PurgeContainer stops and removes container from the docker.
func (p *Pool) PurgeContainer(container *dc.Container) error {
	if err := p.Client.KillContainer(dc.KillContainerOptions{
		ID: container.ID,
	}); err != nil {
		return err
	}

	if err := p.Client.RemoveContainer(dc.RemoveContainerOptions{
		ID:            container.ID,
		Force:         true,
		RemoveVolumes: true,
	}); err != nil {
		return err
	}

	p.rw.Lock()
	p.Containers = p.Containers.Remove(container)
	p.rw.Unlock()

	return nil
}

// GetContainer returns container struct by name.
// Remember to preceed name with `/`. This is how go-dockerclient works.
func (p *Pool) GetContainer(name string) (*dc.Container, bool) {
	for _, container := range p.Containers {
		if container.Name[1:] == name {
			return container, true
		}
	}

	return nil, false
}

// CreateNetwork creates a new network in the docker.
func (p *Pool) CreateNetwork(name string) (*dc.Network, error) {
	net, err := p.Client.CreateNetwork(dc.CreateNetworkOptions{
		Name: name,
	})
	if err != nil {
		return nil, err
	}

	net, err = p.Client.NetworkInfo(net.ID)
	if err != nil {
		return nil, err
	}

	p.rw.Lock()
	p.Networks = append(p.Networks, net)
	p.rw.Unlock()

	return net, nil
}

// PurgeNetwork removes network from the container.
func (p *Pool) PurgeNetwork(net *dc.Network) error {
	err := p.Client.RemoveNetwork(net.ID)
	if err != nil {
		return err
	}

	// Remove `net` from `p.Networks`.
	nets := make([]*dc.Network, 0, len(p.Networks)-1)
	for _, n := range p.Networks {
		if n != net {
			nets = append(nets, n)
		}
	}

	p.rw.Lock()
	p.Networks = nets
	p.rw.Unlock()

	return nil
}

//
// PurgeAll removes every docker resource that was created in a pool.
func (p *Pool) PurgeAll() error {
	var wg sync.WaitGroup
	var errCh chan error

	// Purge containers.
	errCh = make(chan error, len(p.Containers))
	for _, container := range p.Containers {
		wg.Add(1)
		go func(container *dc.Container) {
			defer wg.Done()
			if errPurge := p.PurgeContainer(container); errPurge != nil {
				errCh <- errPurge
			}
		}(container)
	}

	wg.Wait()
	close(errCh)
	if err := <-errCh; err != nil {
		return err
	}

	// Purge networks.
	errCh = make(chan error, len(p.Networks))
	for _, net := range p.Networks {
		wg.Add(1)
		go func(net *dc.Network) {
			defer wg.Done()
			if errPurge := p.PurgeNetwork(net); errPurge != nil {
				errCh <- errPurge
			}
		}(net)
	}

	wg.Wait()
	close(errCh)
	if err := <-errCh; err != nil {
		return err
	}

	return nil
}

// Retry runs `op` every x seconds using exponential back-off strategy.
func Retry(maxWait time.Duration, op func() error) error {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = time.Second * 5
	bo.MaxElapsedTime = maxWait
	return backoff.Retry(op, bo)
}

// GetPort returns a bound host port in the container. `id` is an id of
// the exposed port in the container.
func GetPort(container *dc.Container, id string) string {
	if container.NetworkSettings == nil {
		return ""
	}

	port := dc.Port(id)
	portBinding, ok := container.NetworkSettings.Ports[port]
	if !ok {
		return ""
	} else if len(portBinding) == 0 {
		return ""
	}

	return portBinding[0].HostPort
}

// GetServiceAddr returns a local host with port for the container.
func GetServiceAddr(container *dc.Container, portID string) string {
	return "http://127.0.0.1:" + GetPort(container, portID)
}
