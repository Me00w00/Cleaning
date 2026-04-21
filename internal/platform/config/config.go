package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	AppName     string
	DBPath      string
	LogFilePath string
	Migrations  string
}

func Load() (Config, error) {
	root, err := os.Getwd()
	if err != nil {
		return Config{}, fmt.Errorf("get working directory: %w", err)
	}

	return Config{
		AppName:     "Cleaning Agency",
		DBPath:      filepath.Join(root, "db", "cleaning.sqlite"),
		LogFilePath: filepath.Join(root, "logs", "app.log"),
		Migrations:  filepath.Join(root, "db", "migrations"),
	}, nil
}
