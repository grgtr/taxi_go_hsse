package config

import (
	"github.com/go-yaml/yaml"
	"log"
	"os"
)

type Config struct {
	StoragePath string `yaml:"storage_path" env-required:"true"`
	ConfigDB    DatabaseConfig
}

type DatabaseConfig struct {
	DSN            string `yaml:"dsn"`
	MigrationsPath string `yaml:"migrations_path"`
}

func NewConfig(configPath string) (cfg *Config, err error) {
	bytes, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatal("configPath variable is not set")
		return
	}

	cfg = &Config{}
	err = yaml.Unmarshal(bytes, cfg)
	if err != nil {
		log.Fatal("Couldn't unmarshal config file")
	}
	return
}
