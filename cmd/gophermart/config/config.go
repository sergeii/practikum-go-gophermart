package config

import (
	"time"
)

type Config struct {
	ServerListenAddr       string `env:"RUN_ADDRESS" envDefault:"localhost:8000"`
	ServerShutdownTimeout  time.Duration
	ServerReadTimeout      time.Duration
	ServerWriteTimeout     time.Duration
	DatabaseDSN            string `env:"DATABASE_URI" envDefault:"postgres://gophermart@localhost:5432/gophermart?sslmode=disable"` // nolint: lll
	DatabaseConnectTimeout time.Duration
	AccrualSystemURL       string `env:"ACCRUAL_SYSTEM_ADDRESS" envDefault:"http://localhost:8081"`
	AccrualQueueSize       int
	SecretKeyEncoded       string `env:"SECRET_KEY"`
	SecretKey              []byte
	LogLevel               string
	LogOutput              string
	Production             bool
}
