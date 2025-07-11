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

	"database/sql"

	"github.com/kataras/iris/v12"
	"github.com/rs/zerolog"
)

// main is the entry point for the Tezos Delegation service.
// It sets up configuration, database, services, HTTP server, poller, and graceful shutdown.
func main() {
	// --- Logger Setup ---
	logger := setupLogger()

	// --- Config Load ---
	cfg := mustLoadConfig(logger)

	// --- Database Init ---
	dbConn := mustInitDB(cfg, logger)
	defer dbConn.Close()

	// --- Service and Handler Wiring ---
	delegationRepo := db.NewDelegationRepository(dbConn)
	pollerService := services.NewPoller(delegationRepo, logger)
	delegationService := services.NewDelegationService(delegationRepo, logger)
	delegationHandler := api.NewDelegationHandler(delegationService, logger)

	// --- HTTP Server Setup ---
	app := setupHTTPServer(delegationHandler)

	// --- Signal Handling ---
	quit := setupSignalHandler()

	// --- Poller Start ---
	pollerCtx, cancelPoller := context.WithCancel(context.Background())
	defer cancelPoller()
	pollerService.Start(pollerCtx)

	// --- HTTP Server Start ---
	go startHTTPServer(app, cfg.ServerPort, logger)

	// --- Graceful Shutdown ---
	waitForShutdown(quit, app, pollerService, cancelPoller, logger)
}

func setupLogger() zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339
	return zerolog.New(os.Stdout).With().Timestamp().Logger()
}

func mustLoadConfig(logger zerolog.Logger) *config.Config {
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal().Err(err).Msg("Load config error")
	}
	return cfg
}

func mustInitDB(cfg *config.Config, logger zerolog.Logger) *sql.DB {
	const maxRetries = 10
	const retryDelay = 1 * time.Second

	var lastErr error
	attempt := 0
	for {
		dbConn, err := db.NewDBConnectionFromDSN(cfg.DBUrl)
		if err == nil {
			logger.Info().Int("attempt", attempt).Str("ssl_mode", cfg.SSLMode).Msg("Database connection established successfully")
			return dbConn
		}

		logger.Warn().Err(err).Int("attempt", attempt).Int("maxRetries", maxRetries).Str("dsn", cfg.GetMaskedDBUrl()).Msg("Database connection attempt failed")

		if attempt <= maxRetries {
			logger.Info().Int("attempt", attempt).Dur("delay", retryDelay).Msg("Retrying database connection")
			time.Sleep(retryDelay)
			attempt++
			continue
		}
		logger.Fatal().Err(lastErr).Msg("Database connection error after all retries")

	}

	// This should never be reached
}

func setupHTTPServer(delegationHandler *api.DelegationHandler) *iris.Application {
	app := iris.New()
	api.RegisterRoutes(app, delegationHandler)
	return app
}

func setupSignalHandler() chan os.Signal {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	return quit
}

func startHTTPServer(app *iris.Application, port string, logger zerolog.Logger) {
	if err := app.Listen(":"+port, iris.WithoutInterruptHandler); err != nil {
		logger.Fatal().Err(err).Msg("HTTP server error")
	}
}

func waitForShutdown(quit <-chan os.Signal, app *iris.Application, pollerService *services.PollerService, cancelPoller context.CancelFunc, logger zerolog.Logger) {
	<-quit
	app.Logger().Info("Shutting down server...")

	// Stop poller and wait for completion
	logger.Info().Msg("Shutting down poller")
	cancelPoller()
	done := make(chan struct{})
	go func() {
		pollerService.Wait()
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
