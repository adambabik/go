package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"
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

// NewConfig returns a new config object.
func NewConfig(debug bool) *Config {
	return &Config{Debug: debug}
}

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
func UpdateFromEnv(c interface{}) {
	cv := reflect.ValueOf(c).Elem()

	for i := 0; i < cv.NumField(); i++ {
		field := cv.Field(i)
		jsonTag := cv.Type().Field(i).Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		loadFromEnv(field, tagName(jsonTag))
	}
}

func tagName(tag string) string {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx]
	}
	return tag
}

func loadFromEnv(field reflect.Value, name string) {
	value, ok := os.LookupEnv(strings.ToUpper(name))
	if !ok {
		return
	}

	switch field.Interface().(type) {
	case bool:
		valBool, err := strconv.ParseBool(value)
		if err != nil {
			panic(err)
		}
		field.SetBool(valBool)
	case int:
		valInt, err := strconv.ParseInt(value, 0, strconv.IntSize)
		if err != nil {
			panic(err)
		}
		field.SetInt(valInt)
	case float64:
		valFloat, err := strconv.ParseFloat(value, 64)
		if err != nil {
			panic(err)
		}
		field.SetFloat(valFloat)
	case string:
		field.SetString(value)
	default:
		panic(fmt.Sprintf("Invalid field type: %T", field.Interface()))
	}
}
