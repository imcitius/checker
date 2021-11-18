package config

import (
	"github.com/pkg/errors"
	"log"
)

type EnvConfig struct {
	config string
}

func EnvProvider(config string) *EnvConfig {
	return &EnvConfig{config}
}

func (r *EnvConfig) ReadBytes() ([]byte, error) {
	if r.config == "" {
		log.Fatalf("ENV config empty")
	}
	return []byte(r.config), nil
}

// Read returns the raw bytes for parsing.
func (r *EnvConfig) Read() (map[string]interface{}, error) {
	return nil, errors.New("buf provider does not support this method")
}

// Watch is not supported.
func (r *EnvConfig) Watch(_ func(event interface{}, err error)) error {
	return errors.New("consul provider does not support this method")
}
