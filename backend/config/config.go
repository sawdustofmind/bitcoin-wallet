package config

import (
	"fmt"
	"os"
)

type Config struct {
	XPUB    string
	DB      DBConfig
	Bitcoin BitcoinConfig
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

type BitcoinConfig struct {
	RPCHost string
	RPCUser string
	RPCPass string
}

func Load() (*Config, error) {
	xpub := os.Getenv("XPUB")
	if xpub == "" {
		return nil, fmt.Errorf("XPUB environment variable is required")
	}

	return &Config{
		XPUB: xpub,
		DB: DBConfig{
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASS"),
			Name:     os.Getenv("DB_NAME"),
		},
		Bitcoin: BitcoinConfig{
			RPCHost: os.Getenv("BITCOIN_RPC_HOST"),
			RPCUser: os.Getenv("BITCOIN_RPC_USER"),
			RPCPass: os.Getenv("BITCOIN_RPC_PASS"),
		},
	}, nil
}
