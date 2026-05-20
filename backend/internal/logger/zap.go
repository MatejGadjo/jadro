package logger

import (
	"os"

	"go.uber.org/zap"
)

type Logger struct {
	*zap.SugaredLogger
}

func NewLogger() (*Logger, error) {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	var cfg zap.Config

	if env == "development" {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
	}

	base, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	return &Logger{
		SugaredLogger: base.Sugar(),
	}, nil
}
