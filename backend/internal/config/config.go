// Package config loads and validates application configuration from
// environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime configuration for the API service.
type Config struct {
	AppEnv             string
	HTTPPort           string
	MongoDBURI         string
	MongoDBDatabase    string
	SessionSecret      string
	CookieSecure       bool
	MaxAudioSizeBytes  int64
	CORSAllowedOrigins []string
}

const (
	defaultHTTPPort          = "8080"
	defaultMongoDBURI        = "mongodb://localhost:27017"
	defaultMongoDBDatabase   = "flashcard"
	defaultMaxAudioSizeBytes = 10 * 1024 * 1024 // 10 MB
)

// Load reads configuration from environment variables, applies safe
// defaults for local development, and validates required values.
//
// AppEnv defaults to "development". In that environment only, a missing
// SessionSecret is filled with an insecure development placeholder so the
// service can start without extra setup; every other environment requires
// SESSION_SECRET to be set explicitly.
func Load() (Config, error) {
	cfg := Config{
		AppEnv:             getEnv("APP_ENV", "development"),
		HTTPPort:           getEnv("HTTP_PORT", defaultHTTPPort),
		MongoDBURI:         getEnv("MONGODB_URI", defaultMongoDBURI),
		MongoDBDatabase:    getEnv("MONGODB_DATABASE", defaultMongoDBDatabase),
		SessionSecret:      os.Getenv("SESSION_SECRET"),
		CORSAllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")),
	}

	cookieSecure, err := getBoolEnv("COOKIE_SECURE", cfg.AppEnv == "production")
	if err != nil {
		return Config{}, err
	}
	cfg.CookieSecure = cookieSecure

	maxAudioSize, err := getInt64Env("MAX_AUDIO_SIZE_BYTES", defaultMaxAudioSizeBytes)
	if err != nil {
		return Config{}, err
	}
	cfg.MaxAudioSizeBytes = maxAudioSize

	if cfg.SessionSecret == "" {
		if cfg.AppEnv != "development" {
			return Config{}, fmt.Errorf("SESSION_SECRET is required when APP_ENV=%q", cfg.AppEnv)
		}
		cfg.SessionSecret = "insecure-development-secret-do-not-use-in-production"
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) validate() error {
	if c.HTTPPort == "" {
		return fmt.Errorf("HTTP_PORT must not be empty")
	}
	if c.MongoDBURI == "" {
		return fmt.Errorf("MONGODB_URI must not be empty")
	}
	if c.MongoDBDatabase == "" {
		return fmt.Errorf("MONGODB_DATABASE must not be empty")
	}
	if c.MaxAudioSizeBytes <= 0 {
		return fmt.Errorf("MAX_AUDIO_SIZE_BYTES must be a positive number")
	}
	if len(c.CORSAllowedOrigins) == 0 {
		return fmt.Errorf("CORS_ALLOWED_ORIGINS must not be empty")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) (bool, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("invalid value for %s: %w", key, err)
	}
	return parsed, nil
}

func getInt64Env(key string, fallback int64) (int64, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid value for %s: %w", key, err)
	}
	return parsed, nil
}

func splitCSV(v string) []string {
	if v == "" {
		return nil
	}
	var result []string
	for _, part := range strings.Split(v, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
