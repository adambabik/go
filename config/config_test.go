package config

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateConfig(t *testing.T) {
	c := NewConfig(true)
	assert.True(t, c.Debug)
}

func TestCreateConfigFromFile(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "test.json")
	if assert.NoError(t, err) {
		err := ioutil.WriteFile(tmpfile.Name(), []byte(`{"debug":true}`), 0755)
		if assert.NoError(t, err) {
			c := NewConfigFromFile(tmpfile.Name())
			assert.True(t, c.Debug)
			assert.Equal(t, tmpfile.Name(), c.ConfigFilePath)
		}
	}
}

func TestCreateConfigFromEmptyFile(t *testing.T) {
	assert.NotPanics(t, func() {
		c := NewConfigFromFile("")
		assert.False(t, c.Debug)
	})
}

func TestCreateConfigFromNotExistingFile(t *testing.T) {
	assert.Panics(t, func() {
		NewConfigFromFile("/a/b/c")
	})
}

func TestUpdateConfigFromEnv(t *testing.T) {
	os.Setenv("DEBUG", "1")
	c := NewConfig(false)
	UpdateFromEnv(c)
	assert.True(t, c.Debug)
}

func TestEmbeddedConfigFromEnv(t *testing.T) {
	os.Setenv("A", "1")
	os.Setenv("B", "2.5")
	os.Setenv("C", "c value")
	os.Setenv("D", "3")

	c := &struct {
		*Config
		A int     `json:"-"`
		B float64 `json:"b,omitempty"`
		C string  `json:"c"`
		D int     `json:"d"`
		E string  `json:"e"` // missing in env
	}{
		Config: NewConfig(true),
	}
	UpdateFromEnv(c)
	assert.True(t, c.Debug)
	assert.Empty(t, c.ConfigFilePath)
	assert.Equal(t, 0, c.A)
	assert.Equal(t, 2.5, c.B)
	assert.Equal(t, "c value", c.C)
	assert.Equal(t, 3, c.D)
	assert.Equal(t, "", c.E)
}

func TestEmbeddedConfigFromEnvWithUnsupportedType(t *testing.T) {
	os.Setenv("A", "1")
	c := &struct {
		*Config
		A []byte `json:"a"`
	}{
		Config: NewConfig(true),
	}
	assert.Panics(t, func() {
		UpdateFromEnv(c)
	})
}
