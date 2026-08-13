package platform

import (
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
)

type Config struct {
	HTTPPort      uint16        `env:"HTTP_PORT" envDefault:"8080" validate:"gt=0"`
	MongoURI      string        `env:"MONGO_URI,notEmpty" validate:"url,startswith=mongodb://|startswith=mongodb+srv://"`
	MongoDatabase string        `env:"MONGO_DATABASE,notEmpty"`
	JWTSecret     string        `env:"JWT_SECRET,notEmpty" validate:"min=32"`
	JWTTTL        time.Duration `env:"JWT_TTL,notEmpty" validate:"gt=0"`
}

func LoadConfig() (Config, error) {
	config, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, err
	}
	if err := validator.New().Struct(config); err != nil {
		return Config{}, err
	}
	return config, nil
}
