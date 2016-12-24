package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strconv"
)

type (
	// Config represents an app configuration.
	Config struct {
		// Debug
		Debug bool `json:"debug"`

		// ConfigFilePath stores a file path where the config was read from.
		ConfigFilePath string `json:"config_filepath"`
	}
)

var (
	// DefaultConfig is a default config object.
	DefaultConfig = Config{
		Debug: false,
	}
)

// NewConfigFromFile returns a Config object read from a filename.
func NewConfigFromFile(filename string) *Config {
	config := DefaultConfig

	if filename == "" {
		return &config
	}

	file, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(file, &config)
	if err != nil {
		panic(err)
	}

	config.ConfigFilePath = filename

	return &config
}

// UpdateFromEnv reads config properties from env variables.
// It's safer to load sensitive data from env instead of a file.
func (c *Config) UpdateFromEnv() {
	if debug, ok := os.LookupEnv("DEBUG"); ok {
		debugBool, _ := strconv.ParseBool(debug)
		c.Debug = debugBool
	}
}
