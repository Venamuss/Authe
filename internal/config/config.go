package config

import (
	"log/slog"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Env      string      `env:"ENV" env-default:"development"`
	HTTP     HTTPConfig  `env-prefix:"HTTP_"`
	GRPC     GRPCConfig  `env-prefix:"GRPC_"`
	DB       DBConfig    `env-prefix:"DB_"`
	Redis    RedisConfig `env-prefix:"REDIS_"`
	Security SecurityConfig
}
type HTTPConfig struct {
	Port         string        `env:"PORT" env-default:"8080"`
	ReadTimeout  time.Duration `env:"READ_TIMEOUT" env-default:"5s"`
	WriteTimeout time.Duration `env:"WRITE_TIMEOUT" env-default:"10s"`
	IdleTimeout  time.Duration `env:"IDLE_TIMEOUT" env-default:"60s"`
}
type DBConfig struct {
	URL          string        `env:"URL" env-required:"true"`
	MaxConns     int32         `env:"MAX_CONNS" env-default:"20"`
	MaxIdleConns int32         `env:"MAX_IDLE_CONNS" env-default:"5"`
	MaxConnIdle  time.Duration `env:"MAX_CONN_IDLE_TIME" env-default:"30m"`
}
type RedisConfig struct {
	URL string `env:"URL" env-required:"true"`
}
type SecurityConfig struct {
	JWTSecret string        `env:"JWT_SECRET" env-required:"true"`
	TokenTTL  time.Duration `env:"TOKEN_TTL" env-default:"24h"`
	RateLimit int           `env:"RATE_LIMIT" env-default:"5"`
}
type GRPCConfig struct {
	Port string `env:"PORT" env-default:":50051"`
}

func MustLoad() *Config {
	_ = godotenv.Load()
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		slog.Error("failed to load configuration", "err", err)
		os.Exit(1)
	}
	return &cfg
}
