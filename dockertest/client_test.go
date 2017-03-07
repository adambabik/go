package dockertest

import (
	"errors"
	"fmt"
	"testing"
	"time"

	dc "github.com/fsouza/go-dockerclient"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	testLocalImage            = "adambabik/go-collections:latest"
	testRemoteImageWithoutTag = "sylvainlasnier/echo"
	testRemoteImage           = testRemoteImageWithoutTag + ":latest"
)

func TestCreateNewPool(t *testing.T) {
	Convey("Given a new pool with an empty endpoint", t, func() {
		_, err := NewPool("")

		Convey("Should report no error", func() {
			So(err, ShouldBeNil)
		})
	})
}

func TestRunContainer(t *testing.T) {
	Convey("Given a new poll with an empty endpoint", t, func() {
		pool, err := NewPool("")
		So(err, ShouldBeNil)

		Convey("When starting a container with an unknown local image", func() {
			_, err := pool.RunContainer("some/image", nil, false)

			Convey("Should report an error", func() {
				So(err, ShouldNotBeNil)
			})
		})

		Convey("When starting a container with a local image", func() {
			container, err := pool.RunContainerWithOpts(dc.CreateContainerOptions{
				Config: &dc.Config{
					Image: testLocalImage,
					Cmd:   []string{"-- /bin/echo test"},
				},
				HostConfig: &dc.HostConfig{AutoRemove: true},
			})

			Convey("Should run it successfully", func() {
				So(err, ShouldBeNil)
				So(len(pool.Containers), ShouldEqual, 1)
			})

			Reset(func() {
				pool.Client.RemoveContainer(dc.RemoveContainerOptions{
					ID:    container.ID,
					Force: true,
				})
			})
		})

		Convey("When starting a container with a remote image", func() {
			if testing.Short() {
				fmt.Println("Skipping")
				return
			}

			// Remove the image if already exist.
			pool.Client.RemoveImageExtended(testRemoteImage, dc.RemoveImageOptions{
				Force: true,
			})

			container, err := pool.RunContainer(testRemoteImage, nil, true)

			Convey("Should run it successfully", func() {
				So(err, ShouldBeNil)
				So(container.Config.Image, ShouldEqual, testRemoteImage)
				So(len(pool.Containers), ShouldEqual, 1)
			})

			Reset(func() {
				pool.Client.RemoveContainer(dc.RemoveContainerOptions{
					ID:    container.ID,
					Force: true,
				})
				pool.Client.RemoveImageExtended(testRemoteImage, dc.RemoveImageOptions{
					Force: true,
				})
			})
		})
	})
}

func TestPullPublicImage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TestPullPublicImage")
	}

	Convey("Given a new pool", t, func() {
		pool, err := NewPool("")
		So(err, ShouldBeNil)

		// Remove the image if already exist.
		pool.Client.RemoveImageExtended(testRemoteImage, dc.RemoveImageOptions{
			Force: true,
		})
		So(err, ShouldBeNil)

		Convey("When pulling an existing remote image", func() {
			err := pool.PullImage(testRemoteImageWithoutTag)

			Convey("Should not report any errors and clean up", func() {
				So(err, ShouldBeNil)
			})

			Convey("Should delete image", func() {
				err := pool.Client.RemoveImageExtended(testRemoteImage, dc.RemoveImageOptions{
					Force: true,
				})
				So(err, ShouldBeNil)
			})
		})

		Reset(func() {
			pool.Client.RemoveImageExtended(testRemoteImage, dc.RemoveImageOptions{
				Force: true,
			})
		})
	})
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

	Convey("Given a new pool", t, func() {
		pool, err := NewPool("")
		So(err, ShouldBeNil)

		Convey("Should run multiple containers without a problem", func() {
			containers, err := pool.RunMultipleContainers(
				[]dc.CreateContainerOptions{opts, opts},
			)
			So(err, ShouldBeNil)
			So(len(containers), ShouldEqual, 2)
		})
	})
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

	Convey("Given a new pool", t, func() {
		pool, err := NewPool("")
		So(err, ShouldBeNil)

		Convey("When starting a validand invalid containers", func() {
			containers, err := pool.RunMultipleContainers(
				[]dc.CreateContainerOptions{opts1, opts2},
			)

			Convey("Should start the first and report error for the second", func() {
				So(len(containers), ShouldEqual, 1)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "no such image")
			})

			Reset(func() {
				pool.PurgeContainers(containers)
			})
		})
	})
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

	Convey("Given a new pool", t, func() {
		pool, err := NewPool("")
		So(err, ShouldBeNil)

		Convey("When running two containers", func() {
			containers, err := pool.RunMultipleContainers(
				[]dc.CreateContainerOptions{opts, opts},
			)
			So(err, ShouldBeNil)

			Convey("Should purge them successfully", func() {
				err := pool.PurgeContainers(containers)
				So(err, ShouldBeNil)
				So(len(pool.Containers), ShouldEqual, 0)

				apiContainers, err := pool.Client.ListContainers(
					dc.ListContainersOptions{
						All: true,
						Filters: map[string][]string{
							"label": []string{`app="run-purge"`},
						},
					},
				)
				So(err, ShouldBeNil)
				So(len(apiContainers), ShouldEqual, 0)
			})

			Reset(func() {
				pool.PurgeContainers(containers)
			})
		})
	})
}

func testGetContainer(t *testing.T) {
	t.Parallel()

	opts := dc.CreateContainerOptions{
		Name:       "test_get_container",
		Config:     &dc.Config{Image: testLocalImage},
		HostConfig: &dc.HostConfig{AutoRemove: true},
	}

	Convey("Given a new pool", t, func() {
		pool, err := NewPool("")
		So(err, ShouldBeNil)

		Convey("When running a container", func() {
			container, err := pool.RunContainerWithOpts(opts)
			So(err, ShouldBeNil)

			Convey("Should get up-to-date state from the pool", func() {
				foundContainer, ok := pool.GetContainer("test_get_container")
				So(ok, ShouldBeTrue)
				So(container.ID, ShouldEqual, foundContainer.ID)
			})

			Reset(func() {
				pool.PurgeContainer(container)
			})
		})
	})
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
	Convey("Given a new pool", t, func() {
		pool, err := NewPool("")
		So(err, ShouldBeNil)

		Convey("After creating a new network", func() {
			net, err := pool.CreateNetwork("test-net")
			So(err, ShouldBeNil)
			So(len(pool.Networks), ShouldEqual, 1)
			So("test-net", ShouldEqual, pool.Networks[0].Name)

			Convey("Should be able to get network from the pool", func() {
				_, err := pool.Client.NetworkInfo(net.ID)
				So(err, ShouldBeNil)
			})

			Convey("Should be able to purge the network", func() {
				err := pool.PurgeNetwork(net)
				So(err, ShouldBeNil)
				So(len(pool.Networks), ShouldEqual, 0)

				_, err = pool.Client.NetworkInfo(net.ID)
				So(err, ShouldNotBeNil)
			})
		})

		Reset(func() {
			pool.PurgeAll()
		})
	})
}

func TestPurgeAll(t *testing.T) {
	Convey("Given a new pool", t, func() {
		pool, err := NewPool("")
		So(err, ShouldBeNil)

		Convey("After running a container and network", func() {
			_, err = pool.CreateNetwork("test-net")
			So(err, ShouldBeNil)

			_, err = pool.RunContainer(testLocalImage, nil, false)
			So(err, ShouldBeNil)

			So(len(pool.Containers), ShouldEqual, 1)
			So(len(pool.Networks), ShouldEqual, 1)

			Convey("Should purge all successfully", func() {
				err := pool.PurgeAll()
				So(err, ShouldBeNil)
				So(len(pool.Containers), ShouldEqual, 0)
				So(len(pool.Networks), ShouldEqual, 0)
			})
		})
	})
}

func TestGetPort(t *testing.T) {
	Convey("Given a new pool", t, func() {
		pool, err := NewPool("")
		So(err, ShouldBeNil)

		Convey("When running a container", func() {
			container, err := pool.RunContainerWithOpts(dc.CreateContainerOptions{
				Config: &dc.Config{
					Image:        testLocalImage,
					ExposedPorts: map[dc.Port]struct{}{"8888/tcp": {}},
				},
				HostConfig: &dc.HostConfig{
					PortBindings: map[dc.Port][]dc.PortBinding{
						"8888/tcp": []dc.PortBinding{
							dc.PortBinding{HostPort: "8888"},
						},
					},
				},
			})
			So(err, ShouldBeNil)

			Convey("Should get port successfully", func() {
				So(GetPort(container, "8888/tcp"), ShouldNotBeEmpty)
			})

			Convey("Should get addr successfully", func() {
				So(GetServiceAddr(container, "8888/tcp"), ShouldEqual, "http://127.0.0.1:8888")
			})

			Convey("Should clean uo", func() {
				err := pool.PurgeAll()
				So(err, ShouldBeNil)
			})
		})

		Reset(func() {
			pool.PurgeAll()
		})
	})
}

func TestRetry(t *testing.T) {
	Convey("Given a time and zero calls", t, func() {
		now := time.Now().Unix()
		calls := 0

		Convey("When running a retry", func() {
			err := Retry(5*time.Second, func() error {
				calls++
				if time.Now().Unix()-now > 1 {
					return nil
				}
				return errors.New("Not ready")
			})
			So(err, ShouldBeNil)

			Convey("Should increment value", func() {
				So(calls, ShouldBeGreaterThan, 1)
			})
		})
	})
}
