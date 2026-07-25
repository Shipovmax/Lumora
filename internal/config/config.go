// Package config загружает конфигурацию приложения из окружения (.env + переменные окружения).
package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config — типизированная конфигурация всего приложения.
type Config struct {
	Env      string `env:"ENV" envDefault:"dev"`
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`

	Server   ServerConfig
	Worker   WorkerConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	Auth     AuthConfig
}

type ServerConfig struct {
	Host            string        `env:"SERVER_HOST" envDefault:"0.0.0.0"`
	Port            int           `env:"SERVER_PORT" envDefault:"8080"`
	ReadTimeout     time.Duration `env:"SERVER_READ_TIMEOUT" envDefault:"10s"`
	WriteTimeout    time.Duration `env:"SERVER_WRITE_TIMEOUT" envDefault:"10s"`
	ShutdownTimeout time.Duration `env:"SERVER_SHUTDOWN_TIMEOUT" envDefault:"15s"`
}

func (s ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

type WorkerConfig struct {
	Concurrency int `env:"WORKER_CONCURRENCY" envDefault:"10"`
}

type AuthConfig struct {
	JWTSecret       string        `env:"JWT_SECRET,required"`
	AccessTokenTTL  time.Duration `env:"ACCESS_TOKEN_TTL" envDefault:"15m"`
	RefreshTokenTTL time.Duration `env:"REFRESH_TOKEN_TTL" envDefault:"720h"`
}

type PostgresConfig struct {
	Host           string        `env:"POSTGRES_HOST" envDefault:"localhost"`
	Port           int           `env:"POSTGRES_PORT" envDefault:"5432"`
	User           string        `env:"POSTGRES_USER" envDefault:"lumora"`
	Password       string        `env:"POSTGRES_PASSWORD" envDefault:"lumora"`
	Database       string        `env:"POSTGRES_DB" envDefault:"lumora"`
	SSLMode        string        `env:"POSTGRES_SSLMODE" envDefault:"disable"`
	MaxConns       int32         `env:"POSTGRES_MAX_CONNS" envDefault:"10"`
	ConnectTimeout time.Duration `env:"POSTGRES_CONNECT_TIMEOUT" envDefault:"5s"`
}

// DSN собирает строку подключения в формате, понятном pgx.
func (p PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		p.User, p.Password, p.Host, p.Port, p.Database, p.SSLMode,
	)
}

type RedisConfig struct {
	Addr     string `env:"REDIS_ADDR" envDefault:"localhost:6379"`
	Password string `env:"REDIS_PASSWORD" envDefault:""`
	DB       int    `env:"REDIS_DB" envDefault:"0"`
}

// Load читает .env (если файл существует) и переменные окружения в Config.
func Load() (Config, error) {
	// Отсутствие .env — не ошибка: в проде конфигурация приходит через окружение.
	_ = godotenv.Load()

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}
