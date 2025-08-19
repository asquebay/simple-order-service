package config

import (
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config определяет структуру конфигурации всего приложения целиком
type Config struct {
	HTTPServer `yaml:"http_server"`
	Postgres   `yaml:"postgres"`
	Kafka      `yaml:"kafka"`
	Logger     `yaml:"logger"`
}

// HTTPServer содержит конфигурацию для HTTP-сервера
type HTTPServer struct {
	Port    string        `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
}

// Postgres содержит конфигурацию для подключения к базе данных
type Postgres struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	DBName   string `yaml:"db_name"`
	SSLMode  string `yaml:"ssl_mode"`
}

// Kafka содержит конфигурацию для подключения к кафке
type Kafka struct {
	Brokers []string `yaml:"brokers"`
	Topic   string   `yaml:"topic"`
	GroupID string   `yaml:"group_id"`
}

// Logger содержит конфигурацию для логгера
type Logger struct {
	Level string `yaml:"level"`
}

// MustLoad загружает конфигурацию из файла по указанному пути
// в случае ошибки программа завершается с фатальной ошибкой
func MustLoad(configPath string) *Config {
	if configPath == "" {
		log.Fatal("CONFIG_PATH is not set")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file does not exist: %s", configPath)
	}

	var cfg Config

	file, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("failed to read config file: %s", err)
	}

	if err := yaml.Unmarshal(file, &cfg); err != nil {
		log.Fatalf("failed to unmarshal config: %s", err)
	}

	return &cfg
}
