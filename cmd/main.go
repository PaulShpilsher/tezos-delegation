package main

import (
	"tezos-delegation/internal/api"
	"tezos-delegation/internal/config"
	"tezos-delegation/internal/db"
	"tezos-delegation/internal/services"

	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/rs/zerolog"
)

func main() {
	// Initialize zerolog logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal().Err(err).Msg("Load config error")
	}

	// Initialize database connection
	dbConn, err := db.NewDBConnectionFromDSN(cfg.DBUrl)
	if err != nil {
		logger.Fatal().Err(err).Msg("Database connection error")
	}
	defer dbConn.Close()

	// Initialize repositories and services
	delegationRepo := db.NewDelegationRepository(dbConn)
	ctx, cancelPoller := context.WithCancel(context.Background())
	defer cancelPoller()

	// Start polling service
	poller := services.NewPoller(delegationRepo, logger)
	poller.Start(ctx)

	// Create and configure the Iris application
	app := iris.New()
	delegationService := services.NewDelegationService(delegationRepo)
	delegationHandler := api.NewDelegationHandler(delegationService, logger)
	api.RegisterRoutes(app, api.RouterDeps{DelegationHandler: delegationHandler})

	// Graceful shutdown setup
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := app.Listen(":"+cfg.ServerPort, iris.WithoutInterruptHandler); err != nil {
			logger.Fatal().Err(err).Msg("HTTP server error")
		}
	}()

	<-quit
	app.Logger().Info("Shutting down server...")

	// Stop poller and wait for completion
	logger.Info().Msg("Shutting down poller")
	cancelPoller()
	done := make(chan struct{})
	go func() {
		poller.Wait()
		close(done)
	}()
	select {
	case <-done:
		logger.Info().Msg("Poller shut down cleanly")
	case <-time.After(5 * time.Second):
		logger.Warn().Msg("WARNING: Poller did not shut down within 5 seconds, forcing exit")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.Shutdown(ctx); err != nil {
		logger.Fatal().Err(err).Msg("HTTP Server forced to shutdown")
	}
}
