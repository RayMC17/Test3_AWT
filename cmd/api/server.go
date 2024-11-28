package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (a *applicationDependencies) serve() error {
	// Define the HTTP server
	apiServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", a.config.port),
		Handler:      a.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(a.logger.Handler(), slog.LevelError),
	}

	// Channel to capture shutdown errors
	shutdownError := make(chan error)

	// Start a background goroutine to handle graceful shutdown
	go func() {
		// Create a channel to listen for interrupt/terminate signals
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit // Block until a signal is received

		// Log the shutdown signal
		a.logger.Info("shutting down server", "signal", s.String())

		// Create a context with a timeout for the shutdown process
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := apiServer.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}

		a.logger.Info("completing background tasks", "address", apiServer.Addr)
		a.wg.Wait()
		shutdownError <- nil
	}()

	// Start the server
	a.logger.Info("starting server", "address", apiServer.Addr, "environment", a.config.environment)
	err := apiServer.ListenAndServe()

	// Check if shutdown was due to an error
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// Wait for graceful shutdown to complete
	err = <-shutdownError
	if err != nil {
		return err
	}

	// Log successful server shutdown
	a.logger.Info("stopped server", "address", apiServer.Addr)
	return nil
}
