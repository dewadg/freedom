package main

import (
	"github.com/dewadg/freedom/internal/app/client"
	"github.com/dewadg/freedom/internal/config"
	"github.com/sirupsen/logrus"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		logrus.WithError(err).Fatal("failed to load config")
	}

	client.Run(cfg)
}
