package config

import "testing"

// clearEnv sets every configuration variable to empty for the duration of
// the test (t.Setenv restores the previous value on cleanup), so Load sees
// them as unset and falls back to its defaults.
func clearEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"APP_ENV", "HTTP_PORT", "MONGODB_URI", "MONGODB_DATABASE",
		"SESSION_SECRET", "COOKIE_SECURE", "MAX_AUDIO_SIZE_BYTES",
		"CORS_ALLOWED_ORIGINS",
	}
	for _, k := range keys {
		t.Setenv(k, "")
	}
}

func TestLoad_DevelopmentDefaults(t *testing.T) {
	clearEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	if cfg.AppEnv != "development" {
		t.Errorf("AppEnv = %q, want %q", cfg.AppEnv, "development")
	}
	if cfg.HTTPPort != defaultHTTPPort {
		t.Errorf("HTTPPort = %q, want %q", cfg.HTTPPort, defaultHTTPPort)
	}
	if cfg.MongoDBURI != defaultMongoDBURI {
		t.Errorf("MongoDBURI = %q, want %q", cfg.MongoDBURI, defaultMongoDBURI)
	}
	if cfg.SessionSecret == "" {
		t.Error("SessionSecret should default to a placeholder in development")
	}
	if cfg.CookieSecure {
		t.Error("CookieSecure should default to false in development")
	}
	if cfg.MaxAudioSizeBytes != defaultMaxAudioSizeBytes {
		t.Errorf("MaxAudioSizeBytes = %d, want %d", cfg.MaxAudioSizeBytes, defaultMaxAudioSizeBytes)
	}
	if len(cfg.CORSAllowedOrigins) != 1 || cfg.CORSAllowedOrigins[0] != "http://localhost:5173" {
		t.Errorf("CORSAllowedOrigins = %v, want [http://localhost:5173]", cfg.CORSAllowedOrigins)
	}
}

func TestLoad_ProductionRequiresSessionSecret(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_ENV", "production")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should fail when SESSION_SECRET is missing in production")
	}
}

func TestLoad_ProductionDefaultsCookieSecureToTrue(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("SESSION_SECRET", "a-real-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}
	if !cfg.CookieSecure {
		t.Error("CookieSecure should default to true in production")
	}
}

func TestLoad_InvalidMaxAudioSize(t *testing.T) {
	clearEnv(t)
	t.Setenv("MAX_AUDIO_SIZE_BYTES", "not-a-number")

	if _, err := Load(); err == nil {
		t.Fatal("Load() should fail for a non-numeric MAX_AUDIO_SIZE_BYTES")
	}
}

func TestLoad_ZeroMaxAudioSizeIsInvalid(t *testing.T) {
	clearEnv(t)
	t.Setenv("MAX_AUDIO_SIZE_BYTES", "0")

	if _, err := Load(); err == nil {
		t.Fatal("Load() should fail when MAX_AUDIO_SIZE_BYTES is zero")
	}
}

func TestLoad_ParsesCORSOriginList(t *testing.T) {
	clearEnv(t)
	t.Setenv("CORS_ALLOWED_ORIGINS", " http://a.example , http://b.example ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}
	want := []string{"http://a.example", "http://b.example"}
	if len(cfg.CORSAllowedOrigins) != len(want) {
		t.Fatalf("CORSAllowedOrigins = %v, want %v", cfg.CORSAllowedOrigins, want)
	}
	for i, o := range want {
		if cfg.CORSAllowedOrigins[i] != o {
			t.Errorf("CORSAllowedOrigins[%d] = %q, want %q", i, cfg.CORSAllowedOrigins[i], o)
		}
	}
}
