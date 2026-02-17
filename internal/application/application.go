package application

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/psds-microservice/data-channel-service/internal/config"
	"github.com/psds-microservice/data-channel-service/internal/database"
	"github.com/psds-microservice/data-channel-service/internal/handler"
	"github.com/psds-microservice/data-channel-service/internal/router"
	"github.com/psds-microservice/data-channel-service/internal/service"
	"gorm.io/gorm"
)

type API struct {
	cfg *config.Config
	srv *http.Server
	db  *gorm.DB
}

func NewAPI(cfg *config.Config) (*API, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if err := database.MigrateUp(cfg.DatabaseURL()); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	db, err := database.Open(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("database: %w", err)
	}

	hub := service.NewDataHub()
	dataSvc := service.NewDataService(db)
	dataHandler := handler.NewDataHandler(hub, dataSvc)
	r := router.New(dataHandler)

	srv := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &API{cfg: cfg, srv: srv, db: db}, nil
}

func (a *API) Run(ctx context.Context) error {
	host := a.cfg.AppHost
	if host == "0.0.0.0" {
		host = "localhost"
	}
	base := "http://" + host + ":" + a.cfg.HTTPPort
	log.Printf("HTTP server listening on %s", a.srv.Addr)
	log.Printf("  Swagger UI:    %s/swagger", base)
	log.Printf("  Swagger spec:  %s/swagger/openapi.json", base)
	log.Printf("  Health:        %s/health", base)
	log.Printf("  Ready:         %s/ready", base)
	log.Printf("  WebSocket:     ws://%s:%s/ws/data/:session_id/:user_id", host, a.cfg.HTTPPort)

	go func() {
		if err := a.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := a.srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}
	return nil
}
