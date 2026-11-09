package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	// TODO: THINK WHICH ARE ESSENTIAL AND WHICH ARE NOT. MAYBE SOME OF THEM CAN BE REMOVED OR MADE OPTIONAL
	HTTPAddr          string
	DatabaseURL       string
	LogLevel          string
	ShutdownTimeout   time.Duration
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration

	DatabaseMaxConns        int32
	DatabaseMinConns        int32
	DatabaseMaxConnLifetime time.Duration
	DatabaseConnectTimeout  time.Duration
	DatabaseQueryTimeout    time.Duration
}

func Load() (Config, error) {
	var cfg Config

	var err error

	cfg.HTTPAddr, err = required("HTTP_ADDR")
	if err != nil {
		return Config{}, err
	}

	cfg.LogLevel, err = required("LOG_LEVEL")
	if err != nil {
		return Config{}, err
	}

	cfg.ShutdownTimeout, err = duration("SHUTDOWN_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.DatabaseURL, err = required("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}

	cfg.DatabaseMaxConns, err = int32Env("DATABASE_MAX_CONNS")
	if err != nil {
		return Config{}, err
	}

	cfg.DatabaseMinConns, err = int32Env("DATABASE_MIN_CONNS")
	if err != nil {
		return Config{}, err
	}
	if cfg.DatabaseMaxConns <= 0 {
		return Config{}, fmt.Errorf("DATABASE_MAX_CONNS must be positive")
	}
	if cfg.DatabaseMinConns < 0 || cfg.DatabaseMinConns > cfg.DatabaseMaxConns {
		return Config{}, fmt.Errorf("DATABASE_MIN_CONNS must be between 0 and DATABASE_MAX_CONNS")
	}

	cfg.DatabaseMaxConnLifetime, err = duration("DATABASE_MAX_CONN_LIFETIME")
	if err != nil {
		return Config{}, err
	}

	cfg.DatabaseConnectTimeout, err = duration("DATABASE_CONNECT_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.DatabaseQueryTimeout, err = duration("DATABASE_QUERY_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.ReadTimeout, err = duration("READ_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.ReadHeaderTimeout, err = duration("READ_HEADER_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.WriteTimeout, err = duration("WRITE_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.IdleTimeout, err = duration("IDLE_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func required(name string) (string, error) {
	v, ok := os.LookupEnv(name)
	if !ok || v == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return v, nil
}

func int32Env(name string) (int32, error) {
	v, err := required(name)
	if err != nil {
		return 0, err
	}
	n, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s must be a 32-bit integer: %w", name, err)
	}
	return int32(n), nil
}

func duration(name string) (time.Duration, error) {
	v, err := required(name)
	if err != nil {
		return 0, err
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration like 10s, 5m: %w", name, err)
	}
	return d, nil
}
