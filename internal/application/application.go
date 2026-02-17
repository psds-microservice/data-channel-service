package application

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/psds-microservice/data-channel-service/internal/config"
	"github.com/psds-microservice/data-channel-service/internal/database"
	grpcserver "github.com/psds-microservice/data-channel-service/internal/grpc"
	"github.com/psds-microservice/data-channel-service/internal/handler"
	"github.com/psds-microservice/data-channel-service/internal/service"
	"github.com/psds-microservice/data-channel-service/pkg/constants"
	"github.com/psds-microservice/data-channel-service/pkg/gen/data_channel_service"
	httpSwagger "github.com/swaggo/http-swagger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gorm.io/gorm"
)

// serveOpenAPISpec отдаёт api/openapi.json или api/openapi.swagger.json (из proto: make proto-openapi).
func serveOpenAPISpec() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		for _, path := range []string{"api/openapi.swagger.json", "api/openapi.json", "openapi.json"} {
			data, err := os.ReadFile(path)
			if err == nil {
				w.Header().Set("Content-Type", "application/json")
				w.Write(data)
				return
			}
		}
		exe, _ := os.Executable()
		if exe != "" {
			dir := filepath.Dir(exe)
			for _, name := range []string{"openapi.swagger.json", "openapi.json"} {
				data, err := os.ReadFile(filepath.Join(dir, "api", name))
				if err == nil {
					w.Header().Set("Content-Type", "application/json")
					w.Write(data)
					return
				}
			}
		}
		http.Error(w, "openapi.json not found. Run: make proto-openapi", http.StatusNotFound)
	}
}

// API приложение: HTTP + gRPC серверы (режим api).
type API struct {
	cfg     *config.Config
	httpSrv *http.Server
	grpcSrv *grpc.Server
	lis     net.Listener
	db      *gorm.DB
}

// NewAPI создаёт приложение для режима api.
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

	grpcAddr := cfg.AppHost + ":" + cfg.GRPCPort
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return nil, fmt.Errorf("grpc listen %s: %w (порт занят — остановите другой процесс или задайте GRPC_PORT в .env)", grpcAddr, err)
	}
	grpcSrv := grpc.NewServer()
	grpcImpl := grpcserver.NewServer(grpcserver.Deps{
		Data: dataSvc,
	})
	data_channel_service.RegisterDataChannelServiceServer(grpcSrv, grpcImpl)
	reflection.Register(grpcSrv)

	gatewayMux := runtime.NewServeMux()
	if err := data_channel_service.RegisterDataChannelServiceHandlerServer(context.Background(), gatewayMux, grpcImpl); err != nil {
		return nil, fmt.Errorf("register grpc-gateway: %w", err)
	}

	// Gin router для WebSocket
	ginRouter := gin.New()
	ginRouter.Use(gin.Recovery())
	wsHandler := handler.NewWebSocketHandler(hub, dataSvc)
	ginRouter.GET("/ws/data/:session_id/:user_id", wsHandler.ServeWS)

	// Основной HTTP mux: health/ready/swagger через net/http, REST через grpc-gateway, WebSocket через Gin
	mux := http.NewServeMux()
	mux.HandleFunc(constants.PathHealth, handler.Health)
	mux.HandleFunc(constants.PathReady, handler.Ready)
	mux.HandleFunc(constants.PathSwagger+"/openapi.json", serveOpenAPISpec())
	mux.Handle(constants.PathSwagger+"/", httpSwagger.Handler(
		httpSwagger.URL("openapi.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("list"),
	))
	// WebSocket через Gin
	mux.Handle("/ws/", ginRouter)
	// POST /data/file с multipart/form-data — отдельный handler для совместимости с тестами и клиентами
	uploadMultipart := handler.UploadFileMultipart(dataSvc)
	dataFileHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/data/file" && r.Method == http.MethodPost &&
			strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "multipart/form-data") {
			uploadMultipart(w, r)
			return
		}
		gatewayMux.ServeHTTP(w, r)
	})
	mux.Handle("/", dataFileHandler)

	httpAddr := cfg.AppHost + ":" + cfg.HTTPPort
	httpSrv := &http.Server{
		Addr:              httpAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &API{
		cfg:     cfg,
		httpSrv: httpSrv,
		grpcSrv: grpcSrv,
		lis:     lis,
		db:      db,
	}, nil
}

// Run запускает HTTP и gRPC серверы, блокируется до отмены ctx.
func (a *API) Run(ctx context.Context) error {
	httpAddr := a.httpSrv.Addr
	grpcAddr := a.lis.Addr().String()
	host := a.cfg.AppHost
	if host == "0.0.0.0" {
		host = "localhost"
	}
	base := "http://" + host + ":" + a.cfg.HTTPPort
	log.Printf("HTTP server listening on %s", httpAddr)
	log.Printf("  Swagger UI:    %s/swagger", base)
	log.Printf("  Swagger spec:  %s/swagger/openapi.json", base)
	log.Printf("  Health:        %s/health", base)
	log.Printf("  Ready:         %s/ready", base)
	log.Printf("  WebSocket:     ws://%s:%s/ws/data/:session_id/:user_id", host, a.cfg.HTTPPort)
	log.Printf("  REST API:      %s/data/", base)
	log.Printf("gRPC server listening on %s", grpcAddr)
	log.Printf("  gRPC endpoint: %s (reflection enabled)", grpcAddr)

	go func() {
		if err := a.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http: %v", err)
		}
	}()

	go func() {
		if err := a.grpcSrv.Serve(a.lis); err != nil {
			log.Printf("grpc: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := a.httpSrv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}
	a.grpcSrv.GracefulStop()
	return nil
}
