package main
// PR Edit 1
import (
	"context"
	"log"
	"net/http"
	"op-status/api"
	"op-status/config"
	"op-status/pkg"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)

	// Create scheduler and start metrics collection
	scheduler := pkg.NewScheduler()
	scheduler.Start()
	defer scheduler.Stop()
	log.Println("Metrics collection scheduler started")

	// Create API handler
	handler := api.NewHandler(scheduler)

	// Initialize Gin router
	router := gin.Default()

	// Register routes
	handler.RegisterRoutes(router)

	// Create HTTP server
	srv := &http.Server{
		Addr:         config.APIPort,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("API server starting on %s", config.APIPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %s", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	log.Println("Shutting down server...")

	// Give 5 seconds for existing connections to finish
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %s", err)
	}

	log.Println("Server exiting")
}
