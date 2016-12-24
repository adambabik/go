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
	config := config.NewConfigFromFile(*configFilename)
	config.UpdateFromEnv()

	e := echo.New()
	e.Debug = config.Debug

	e.Use(middleware.Logger())

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

	e.Logger.Info("Running server on :8080")
	e.Logger.Fatal(e.Start(":8080"))
}