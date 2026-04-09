package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/doublemo/nats-ui/internal/config"
	"github.com/doublemo/nats-ui/internal/handlers"
	"github.com/doublemo/nats-ui/internal/service"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	manager, err := service.NewConnectionManager(cfg)
	if err != nil {
		log.Fatalf("init connection manager failed: %v", err)
	}
	defer manager.Close()

	natsService := service.NewNATSService(manager)
	natsHandler := handlers.NewNATSHandler(natsService, manager)

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), corsMiddleware())
	natsHandler.Register(router)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("nats-ui backend started on %s", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("start http server failed: %v", err)
		}
	}()

	waitShutdown(server)
}

func waitShutdown(server *http.Server) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown http server failed: %v", err)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
