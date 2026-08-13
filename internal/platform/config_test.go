package platform

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	environment := validEnvironment()
	environment["HTTP_PORT"] = "9090"
	setEnvironment(t, environment)

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	if config.HTTPPort != 9090 {
		t.Errorf("HTTPPort = %d, want 9090", config.HTTPPort)
	}
	if config.MongoURI != environment["MONGO_URI"] {
		t.Errorf("MongoURI = %q, want %q", config.MongoURI, environment["MONGO_URI"])
	}
	if config.MongoDatabase != environment["MONGO_DATABASE"] {
		t.Errorf("MongoDatabase = %q, want %q", config.MongoDatabase, environment["MONGO_DATABASE"])
	}
	if config.JWTSecret != environment["JWT_SECRET"] {
		t.Errorf("JWTSecret = %q, want configured value", config.JWTSecret)
	}
	if config.JWTTTL != time.Hour {
		t.Errorf("JWTTTL = %s, want 1h", config.JWTTTL)
	}
}

func TestLoadConfigUsesDefaultHTTPPort(t *testing.T) {
	setEnvironment(t, validEnvironment())
	unsetEnvironment(t, "HTTP_PORT")

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	if config.HTTPPort != 8080 {
		t.Errorf("HTTPPort = %d, want 8080", config.HTTPPort)
	}
}

func TestLoadConfigRejectsMissingOrInvalidValues(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		value     string
		wantError string
	}{
		{name: "invalid HTTP port", key: "HTTP_PORT", value: "70000", wantError: "HTTPPort"},
		{name: "missing Mongo URI", key: "MONGO_URI", value: "", wantError: "MONGO_URI"},
		{name: "invalid Mongo URI", key: "MONGO_URI", value: "http://localhost:27017", wantError: "MongoURI"},
		{name: "missing Mongo database", key: "MONGO_DATABASE", value: "", wantError: "MONGO_DATABASE"},
		{name: "missing JWT secret", key: "JWT_SECRET", value: "", wantError: "JWT_SECRET"},
		{name: "short JWT secret", key: "JWT_SECRET", value: "too-short", wantError: "JWTSecret"},
		{name: "missing JWT TTL", key: "JWT_TTL", value: "", wantError: "JWT_TTL"},
		{name: "invalid JWT TTL", key: "JWT_TTL", value: "tomorrow", wantError: "JWTTTL"},
		{name: "negative JWT TTL", key: "JWT_TTL", value: "-1h", wantError: "JWTTTL"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			environment := validEnvironment()
			environment[test.key] = test.value
			setEnvironment(t, environment)
			unsetEnvironment(t, "HTTP_PORT")
			if test.key == "HTTP_PORT" {
				t.Setenv(test.key, test.value)
			}

			_, err := LoadConfig()
			if err == nil {
				t.Fatal("loadConfig() error = nil, want validation error")
			}
			if !strings.Contains(err.Error(), test.wantError) {
				t.Errorf("loadConfig() error = %q, want it to contain %q", err, test.wantError)
			}
		})
	}
}

func validEnvironment() map[string]string {
	return map[string]string{
		"MONGO_URI":      "mongodb://localhost:27017",
		"MONGO_DATABASE": "user_management",
		"JWT_SECRET":     "a-local-test-secret-with-32-characters",
		"JWT_TTL":        "1h",
	}
}

func setEnvironment(t *testing.T, environment map[string]string) {
	t.Helper()
	for key, value := range environment {
		t.Setenv(key, value)
	}
}

func unsetEnvironment(t *testing.T, key string) {
	t.Helper()
	value, exists := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("os.Unsetenv(%q) error = %v", key, err)
	}
	t.Cleanup(func() {
		if exists {
			_ = os.Setenv(key, value)
			return
		}
		_ = os.Unsetenv(key)
	})
}
