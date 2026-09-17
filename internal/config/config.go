// Package config regroupe la configuration du client web.
//
// Le client n'a besoin que de deux informations : le port sur lequel il écoute
// et l'adresse du serveur API. Il ne connaît ni la base de données ni le
// secret de signature des jetons : ces éléments sont l'affaire du serveur.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// Config contient les paramètres nécessaires au démarrage du client.
type Config struct {
	Port string

	// APIBaseURL est la racine de l'API, sans slash final.
	APIBaseURL string
}

// Load lit la configuration depuis l'environnement.
//
// Contrairement au serveur, les deux valeurs ont un défaut utilisable en
// développement : le client ne manipule aucun secret, il n'y a donc pas de
// risque à démarrer avec une configuration implicite.
func Load() (*Config, error) {
	cfg := &Config{
		Port:       getEnv("PORT", "3000"),
		APIBaseURL: strings.TrimRight(getEnv("API_BASE_URL", "http://localhost:8080"), "/"),
	}

	// L'adresse de l'API est validée au démarrage. Sans ce contrôle, une
	// URL mal formée ne se manifesterait qu'à la première requête d'un
	// utilisateur, sous la forme d'une erreur peu compréhensible.
	parsed, err := url.Parse(cfg.APIBaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("API_BASE_URL invalide : %q (attendu par exemple http://localhost:8080)", cfg.APIBaseURL)
	}

	return cfg, nil
}

// getEnv retourne la valeur de la variable d'environnement demandée,
// ou la valeur par défaut fournie si celle-ci n'est pas définie.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
