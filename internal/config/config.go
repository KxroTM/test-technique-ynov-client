// Package config regroupe la configuration du client web

package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// Config contient les paramètres nécessaires au démarrage du client
type Config struct {
	Port           string
	APIBaseURL     string
	GoogleClientID string
}

// Load lit la configuration depuis l'environnement
func Load() (*Config, error) {
	cfg := &Config{
		Port:           getEnv("PORT", "3000"),
		APIBaseURL:     strings.TrimRight(getEnv("API_BASE_URL", "http://localhost:8080"), "/"),
		GoogleClientID: os.Getenv("GOOGLE_CLIENT_ID"),
	}

	parsed, err := url.Parse(cfg.APIBaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("API_BASE_URL invalide : %q (attendu par exemple http://localhost:8080)", cfg.APIBaseURL)
	}

	return cfg, nil
}

// GoogleEnabled indique si le bouton de connexion Google doit être proposé
func (c *Config) GoogleEnabled() bool {
	return c.GoogleClientID != ""
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
