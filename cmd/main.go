package main

import (
	"log"
	"tezoz-delegation/internal/api"
	"tezoz-delegation/internal/config"
	"tezoz-delegation/internal/db"
	"tezoz-delegation/internal/services"

	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kataras/iris/v12"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		println("Config error:", err.Error())
		os.Exit(1)
	}

	// Initialize database connection
	dbConn, err := db.NewDBConnectionFromDSN(cfg.DBUrl)
	if err != nil {
		println("Database connection error:", err.Error())
		os.Exit(1)
	}
	defer dbConn.Close()

	// Initialize repositories and services
	delegationRepo := db.NewDelegationRepository(dbConn)
	ctx, cancelPoller := context.WithCancel(context.Background())
	defer cancelPoller()

	// Start polling service
	poller := services.NewPoller(delegationRepo)
	poller.Start(ctx)

	// Create and configure the Iris application
	app := iris.New()
	delegationHandler := api.NewDelegationHandler(delegationRepo)
	api.RegisterRoutes(app, api.RouterDeps{DelegationHandler: delegationHandler})

	// Graceful shutdown setup
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := app.Listen(":"+cfg.ServerPort, iris.WithoutInterruptHandler); err != nil {
			app.Logger().Fatalf("Server error: %v", err)
		}
	}()

	<-quit
	app.Logger().Info("Shutting down server...")

	// Stop poller and wait for completion
	log.Println("Shutting down poller")
	cancelPoller()
	done := make(chan struct{})
	go func() {
		poller.Wait()
		close(done)
	}()
	select {
	case <-done:
		log.Println("Poller shut down cleanly")
	case <-time.After(5 * time.Second):
		log.Println("WARNING: Poller did not shut down within 5 seconds, forcing exit")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.Shutdown(ctx); err != nil {
		app.Logger().Fatalf("Server forced to shutdown: %v", err)
	}
}
