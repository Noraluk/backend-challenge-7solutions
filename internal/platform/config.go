package platform

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHTTPPort     = uint16(8080)
	minimumSecretLength = 32
)

type Config struct {
	HTTPPort      uint16
	MongoURI      string
	MongoDatabase string
	JWTSecret     string
	JWTTTL        time.Duration
}

func LoadConfig() (Config, error) {
	return loadConfig(os.LookupEnv)
}

func loadConfig(lookup func(string) (string, bool)) (Config, error) {
	config := Config{HTTPPort: defaultHTTPPort}

	if value, ok := lookup("HTTP_PORT"); ok && strings.TrimSpace(value) != "" {
		port, err := strconv.ParseUint(value, 10, 16)
		if err != nil || port == 0 {
			return Config{}, fmt.Errorf("HTTP_PORT must be an integer between 1 and 65535")
		}
		config.HTTPPort = uint16(port)
	}

	mongoURI, err := requiredValue(lookup, "MONGO_URI")
	if err != nil {
		return Config{}, err
	}
	parsedMongoURI, err := url.Parse(mongoURI)
	if err != nil || parsedMongoURI.Host == "" || (parsedMongoURI.Scheme != "mongodb" && parsedMongoURI.Scheme != "mongodb+srv") {
		return Config{}, fmt.Errorf("MONGO_URI must be a valid mongodb or mongodb+srv URI")
	}
	config.MongoURI = mongoURI

	config.MongoDatabase, err = requiredValue(lookup, "MONGO_DATABASE")
	if err != nil {
		return Config{}, err
	}

	config.JWTSecret, err = requiredValue(lookup, "JWT_SECRET")
	if err != nil {
		return Config{}, err
	}
	if len(config.JWTSecret) < minimumSecretLength {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least %d characters", minimumSecretLength)
	}

	jwtTTL, err := requiredValue(lookup, "JWT_TTL")
	if err != nil {
		return Config{}, err
	}
	config.JWTTTL, err = time.ParseDuration(jwtTTL)
	if err != nil || config.JWTTTL <= 0 {
		return Config{}, fmt.Errorf("JWT_TTL must be a positive duration such as 1h")
	}

	return config, nil
}

func requiredValue(lookup func(string) (string, bool), key string) (string, error) {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s is required", key)
	}

	return value, nil
}
