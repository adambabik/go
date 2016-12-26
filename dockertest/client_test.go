package dockertest

import (
	"errors"
	"testing"
	"time"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"
)

const (
	testLocalImage            = "adambabik/go-collections:latest"
	testRemoteImageWithoutTag = "sylvainlasnier/echo"
	testRemoteImage           = testRemoteImageWithoutTag + ":latest"
)

func TestCreateNewPool(t *testing.T) {
	_, err := NewPool("")
	assert.Nil(t, err)
}

func TestRunContainerWithoutImage(t *testing.T) {
	pool, err := NewPool("")
	if assert.NoError(t, err) {
		_, err := pool.RunContainer("some/image", nil, false)
		assert.Error(t, err)
	}
}

func TestRunLocalContainer(t *testing.T) {
	pool, err := NewPool("")
	if assert.NoError(t, err) {
		_, err := pool.RunContainerWithOpts(dc.CreateContainerOptions{
			Config: &dc.Config{
				Image: testLocalImage,
				Cmd:   []string{"-- /bin/echo test"},
			},
			HostConfig: &dc.HostConfig{AutoRemove: true},
		})
		assert.Nil(t, err)
		assert.Equal(t, 1, len(pool.Containers))
	}
}

func TestPullPublicImage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TestPullPublicImage")
	}

	pool, err := NewPool("")
	if assert.NoError(t, err) {
		// Remove the image if already exist.
		pool.Client.RemoveImageExtended(testRemoteImage, dc.RemoveImageOptions{
			Force: true,
		})

		err := pool.PullImage(testRemoteImageWithoutTag)
		assert.Nil(t, err)

		// Clean up.
		err = pool.Client.RemoveImageExtended(testRemoteImage, dc.RemoveImageOptions{
			Force: true,
		})
		assert.Nil(t, err)
	}
}

func TestRunContainerWithPulling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TestRunContainerWithPulling")
	}

	pool, err := NewPool("")
	if assert.NoError(t, err) {
		// Remove the image if already exist.
		pool.Client.RemoveImageExtended(testRemoteImage, dc.RemoveImageOptions{
			Force: true,
		})

		container, err := pool.RunContainer(testRemoteImage, nil, true)
		if assert.NoError(t, err) {
			assert.Equal(t, testRemoteImage, container.Config.Image)
			assert.Equal(t, 1, len(pool.Containers))
		}

		// Clean up.
		err = pool.Client.RemoveContainer(dc.RemoveContainerOptions{
			ID:    container.ID,
			Force: true,
		})
		assert.Nil(t, err)
		err = pool.Client.RemoveImageExtended(testRemoteImage, dc.RemoveImageOptions{
			Force: true,
		})
		assert.Nil(t, err)
	}
}

func testRunMultipleContainers(t *testing.T) {
	t.Parallel()

	opts := dc.CreateContainerOptions{
		Config: &dc.Config{
			Image: testLocalImage,
			Cmd:   []string{"-- /bin/echo test"},
		},
		HostConfig: &dc.HostConfig{AutoRemove: true},
	}

	pool, err := NewPool("")
	if assert.NoError(t, err) {
		containers, err := pool.RunMultipleContainers(
			[]dc.CreateContainerOptions{opts, opts},
		)
		if assert.NoError(t, err) {
			assert.Equal(t, 2, len(containers))
		}
	}
}

func testRunMultipleContainersReportsFirstError(t *testing.T) {
	t.Parallel()

	opts1 := dc.CreateContainerOptions{
		Config: &dc.Config{
			Image: testLocalImage,
			Cmd:   []string{"-- /bin/echo test"},
		},
		HostConfig: &dc.HostConfig{AutoRemove: true},
	}
	opts2 := dc.CreateContainerOptions{
		Config:     &dc.Config{Image: "some/non-existing/image"},
		HostConfig: &dc.HostConfig{AutoRemove: true},
	}

	pool, err := NewPool("")
	if assert.NoError(t, err) {
		containers, err := pool.RunMultipleContainers(
			[]dc.CreateContainerOptions{opts1, opts2},
		)
		assert.Equal(t, 1, len(containers))
		assert.NotNil(t, err)
	}
}

func testRunAndPurgeContainers(t *testing.T) {
	t.Parallel()

	opts := dc.CreateContainerOptions{
		Config: &dc.Config{
			Image: testLocalImage,
			Labels: map[string]string{
				"app": "run-purge",
			},
		},
	}

	pool, err := NewPool("")
	if assert.NoError(t, err) {
		containers, err := pool.RunMultipleContainers(
			[]dc.CreateContainerOptions{opts, opts},
		)
		if assert.NoError(t, err) {
			assert.Equal(t, 2, len(containers))

			err := pool.PurgeContainers(containers)
			if assert.NoError(t, err) {
				assert.Equal(t, 0, len(pool.Containers))

				apiContainers, err := pool.Client.ListContainers(
					dc.ListContainersOptions{
						All: true,
						Filters: map[string][]string{
							"label": []string{`app="run-purge"`},
						},
					},
				)
				if assert.NoError(t, err) {
					assert.Equal(t, 0, len(apiContainers))
				}
			}
		}
	}
}

func testGetContainer(t *testing.T) {
	t.Parallel()

	opts := dc.CreateContainerOptions{
		Name:   "test_get_container",
		Config: &dc.Config{Image: testLocalImage},
	}

	pool, err := NewPool("")
	if assert.NoError(t, err) {
		container, err := pool.RunContainerWithOpts(opts)
		if assert.NoError(t, err) {
			foundContainer, ok := pool.GetContainer("test_get_container")
			if assert.True(t, ok) {
				assert.Equal(t, foundContainer.ID, container.ID)
			}
		}

		assert.Nil(t, pool.PurgeContainers(pool.Containers))
	}
}

func TestRunMultipleContainers(t *testing.T) {
	t.Run("RunMultipleContainers", func(t *testing.T) {
		t.Run("testRunMultipleContainers",
			testRunMultipleContainers)
		t.Run("testRunMultipleContainersReportsFirstError",
			testRunMultipleContainersReportsFirstError)
		t.Run("testRunAndPurgeContainers",
			testRunAndPurgeContainers)
		t.Run("testGetContainer",
			testGetContainer)
	})
}

func TestCreateNetwork(t *testing.T) {
	pool, err := NewPool("")
	if assert.NoError(t, err) {
		net, err := pool.CreateNetwork("test-net")
		if assert.NoError(t, err) {
			assert.Equal(t, 1, len(pool.Networks))
			assert.Equal(t, "test-net", pool.Networks[0].Name)

			assert.Nil(t, pool.Client.RemoveNetwork(net.ID))
		}
	}
}

func TestPurgeNetwork(t *testing.T) {
	pool, err := NewPool("")
	if assert.NoError(t, err) {
		net, err := pool.CreateNetwork("test-net")
		if assert.NoError(t, err) {
			assert.Equal(t, 1, len(pool.Networks))

			_, err := pool.Client.NetworkInfo(net.ID)
			if assert.NoError(t, err) {
				err := pool.PurgeNetwork(net)
				if assert.NoError(t, err) {
					assert.Equal(t, 0, len(pool.Networks))

					_, err := pool.Client.NetworkInfo(net.ID)
					assert.Error(t, err)
				}
			}
		}
	}
}

func TestPurgeAll(t *testing.T) {
	pool, err := NewPool("")
	if !assert.NoError(t, err) {
		return
	}

	_, err = pool.CreateNetwork("test-net")
	if !assert.NoError(t, err) {
		return
	}

	_, err = pool.RunContainer(testLocalImage, nil, false)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, 1, len(pool.Containers))
	assert.Equal(t, 1, len(pool.Networks))

	if assert.NoError(t, pool.PurgeAll()) {
		assert.Equal(t, 0, len(pool.Containers))
		assert.Equal(t, 0, len(pool.Networks))
	}
}

func TestGetPort(t *testing.T) {
	pool, err := NewPool("")
	if assert.NoError(t, err) {
		container, err := pool.RunContainer(testLocalImage, nil, false)
		if assert.NoError(t, err) {
			assert.NotEmpty(t, GetPort(container, "8080/tcp"))
			assert.Empty(t, GetPort(container, "8081/tcp"))
		}
		err = pool.PurgeAll()
		assert.NoError(t, err)
	}
}

func TestGetServiceAddr(t *testing.T) {
	pool, err := NewPool("")
	if assert.NoError(t, err) {
		container, err := pool.RunContainerWithOpts(dc.CreateContainerOptions{
			Config: &dc.Config{
				Image:        testLocalImage,
				ExposedPorts: map[dc.Port]struct{}{"8080/tcp": {}},
			},
			HostConfig: &dc.HostConfig{
				PortBindings: map[dc.Port][]dc.PortBinding{
					"8080/tcp": []dc.PortBinding{
						dc.PortBinding{HostPort: "8080"},
					},
				},
			},
		})
		if assert.NoError(t, err) {
			addr := GetServiceAddr(container, "8080/tcp")
			assert.Equal(t, "http://127.0.0.1:8080", addr)
		}
		err = pool.PurgeAll()
		assert.NoError(t, err)
	}
}

func TestRetry(t *testing.T) {
	now := time.Now().Unix()
	calls := 0
	err := Retry(5*time.Second, func() error {
		calls++
		if time.Now().Unix()-now > 1 {
			return nil
		}
		return errors.New("Not ready")
	})
	assert.NoError(t, err)
	assert.True(t, calls > 1)
}
