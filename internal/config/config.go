package config

import (
	"fmt"
	"log"

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

// URL returns a formatted PostgreSQL connection string using the DB configuration values.
func (d *DB) URL() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

type Server struct {
	Port              string `env:"PORT"`                // Server port number
	PageSize          int    `env:"PAGE_SIZE"`           // Integer value for pagination size
	ReadHeaderTimeout int    `env:"READ_HEADER_TIMEOUT"` // Used to limit execution time of the http.Handler
}

type Worker struct {
	PoolSize           int `env:"POOL_SIZE"`            // Integer value for worker pool size
	BatchSize          int `env:"BATCH_SIZE"`           // Integer value for batch processing size
	StuckCheckInterval int `env:"STUCK_CHECK_INTERVAL"` // Integer value for checking stuck jobs interval
}

type Config struct {
	DB     DB     `envPrefix:"DB_"`
	Server Server `envPrefix:"SERVER_"`
	Worker Worker `envPrefix:"WORKER_"`
}

// NewConfig creates and returns a new Config instance by loading environment variables
// from .env file and parsing them into the Config struct. It panics if parsing fails.
func NewConfig() Config {
	if err := godotenv.Load(); err != nil {
		log.Println(".env was not found")
	}

	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		panic(err)
	}

	return cfg
}
