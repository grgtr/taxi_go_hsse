// internal/config/config.go
package config

import (
	"encoding/json"
	"os"
)

// MongoDBConfig represents the MongoDB configuration.
type MongoDBConfig struct {
	URI string `json:"uri"`
}

// Config represents the overall configuration of the client service.
type Config struct {
	IP       string        `json:"ip"`
	Port     string        `json:"port"`
	Version  string        `json:"version"`
	HTTP     string        `json:"ip+port"`
	Database MongoDBConfig `json:"database"`
}

// Parse reads the configuration from a file.
func Parse(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
