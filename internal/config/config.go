package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type DB struct {
	Host     string `env:"HOST"`
	Port     string `env:"PORT"`
	Name     string `env:"NAME"`
	User     string `env:"USER"`
	Password string `env:"PASSWORD"`
	SSLMode  string `env:"SSLMODE"`
}

func (d *DB) URL() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

type Server struct {
	Port     string `env:"PORT"`
	PageSize int    `env:"PAGE_SIZE"`
}

type Worker struct {
	PoolSize           int `env:"POOL_SIZE"`
	BatchSize          int `env:"BATCH_SIZE"`
	StuckCheckInterval int `env:"STUCK_CHECK_INTERVAL"`
}

type Config struct {
	DB     DB     `envPrefix:"DB_"`
	Server Server `envPrefix:"SERVER_"`
	Worker Worker `envPrefix:"WORKER_"`
}

func NewConfig() Config {
	if err := godotenv.Load(); err != nil {
		fmt.Println(err)
	}

	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		panic(err)
	}

	return cfg
}
