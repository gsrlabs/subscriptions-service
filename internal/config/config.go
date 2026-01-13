package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App        AppConfig
	Database   DatabaseConfig
	Migrations MigrationConfig
	Test       TestConfig
}

type AppConfig struct {
	Port string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string

	MaxConns int32
	MinConns int32
}

type MigrationConfig struct {
	Path string
}

type TestConfig struct {
	DBHost                string
	MigrationsPath        string
	HandlerMigrationsPath string
}

func Load(path string) (*Config, error) {
	v := viper.New()

	// --- File ---
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	// --- Env ---
	v.SetEnvPrefix("")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := &Config{}

	cfg.App = AppConfig{
		Port: v.GetString("app.port"),
	}

	cfg.Database = DatabaseConfig{
		Host:     v.GetString("database.host"),
		Port:     v.GetInt("database.port"),
		User:     v.GetString("database.user"),
		Name:     v.GetString("database.name"),
		SSLMode:  v.GetString("database.sslmode"),
		MaxConns: v.GetInt32("database.max_conns"),
		MinConns: v.GetInt32("database.min_conns"),
	}

	// Password is from env ONLY
	cfg.Database.Password = os.Getenv("DB_PASSWORD")

	cfg.Migrations = MigrationConfig{
		Path: v.GetString("migrations.path"),
	}

	cfg.Test = TestConfig{
		DBHost:                v.GetString("test.db_host"),
		MigrationsPath:        v.GetString("test.migrations_path"),
		HandlerMigrationsPath: v.GetString("test.handler_migrations_path"),
	}

	return cfg, nil
}
