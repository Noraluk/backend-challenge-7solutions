package platform

import (
	"strings"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	environment := validEnvironment()
	environment["HTTP_PORT"] = "9090"

	config, err := loadConfig(mapLookup(environment))
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
	config, err := loadConfig(mapLookup(validEnvironment()))
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	if config.HTTPPort != defaultHTTPPort {
		t.Errorf("HTTPPort = %d, want %d", config.HTTPPort, defaultHTTPPort)
	}
}

func TestLoadConfigRejectsMissingOrInvalidValues(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		value     string
		wantError string
	}{
		{name: "invalid HTTP port", key: "HTTP_PORT", value: "70000", wantError: "HTTP_PORT"},
		{name: "missing Mongo URI", key: "MONGO_URI", value: "", wantError: "MONGO_URI is required"},
		{name: "invalid Mongo URI", key: "MONGO_URI", value: "http://localhost:27017", wantError: "MONGO_URI must be"},
		{name: "missing Mongo database", key: "MONGO_DATABASE", value: "", wantError: "MONGO_DATABASE is required"},
		{name: "missing JWT secret", key: "JWT_SECRET", value: "", wantError: "JWT_SECRET is required"},
		{name: "short JWT secret", key: "JWT_SECRET", value: "too-short", wantError: "JWT_SECRET must be"},
		{name: "missing JWT TTL", key: "JWT_TTL", value: "", wantError: "JWT_TTL is required"},
		{name: "invalid JWT TTL", key: "JWT_TTL", value: "tomorrow", wantError: "JWT_TTL must be"},
		{name: "negative JWT TTL", key: "JWT_TTL", value: "-1h", wantError: "JWT_TTL must be"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			environment := validEnvironment()
			environment[test.key] = test.value

			_, err := loadConfig(mapLookup(environment))
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

func mapLookup(environment map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := environment[key]
		return value, ok
	}
}
