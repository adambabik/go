package main

import (
	"flag"
	"net/http"

	"github.com/adambabik/go-collections/config"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

var (
	configFilename = flag.String("config", "", "a filename with config")
)

func init() {
	flag.Parse()
}

func main() {
	c := config.NewConfigFromFile(*configFilename)
	config.UpdateFromEnv(c)

	e := echo.New()
	e.Debug = c.Debug

	e.Use(middleware.Logger())

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

	e.Logger.Info("Running server on :8888")
	e.Logger.Fatal(e.Start(":8888"))
}
