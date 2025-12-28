package app

import (
	"fmt"
	"gophermart/internal/config"
	"gophermart/internal/logger"
	"gophermart/internal/repository"
	"gophermart/internal/service"
)

func Setup() (*service.Service, error) {
	cfg, err := config.NewConfig()
	if err != nil {
		return nil, fmt.Errorf("error create config: %w", err)
	}

	log, err := logger.NewSugaredLogger("info")
	if err != nil {
		return nil, fmt.Errorf("error create logger: %w", err)
	}

	db, err := repository.NewDatabaseStorage(cfg.DBURI, log)
	if err != nil {
		return nil, fmt.Errorf("error create repository: %w", err)
	}

	srvc := service.Service{
		Storage: db,
		Config:  cfg,
		Logger:  log,
	}
	go func() {
		if err := srvc.UpdateOrderStatus(); err != nil {
			log.Infof("Order processor stopped with error: %v", err)
		}
	}()
	return &srvc, nil
}
