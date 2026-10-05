package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"taskapi/internal/cache"
	"taskapi/internal/handler"
	"taskapi/internal/middleware"
	"taskapi/internal/repository"
	"taskapi/internal/service"
)

func main() {
	logger := log.New(os.Stdout, "[taskapi] ", log.LstdFlags|log.Lshortfile)

	repo := repository.NewMemoryTaskRepo()
	appCache := cache.New(30 * time.Second)
	svc := service.NewTaskService(repo, appCache)
	notifier := service.NewNotifier()
	taskHandler := handler.NewTaskHandler(svc, notifier)

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logging)

	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	r.Route("/api/v1", func(r chi.Router) {
		taskHandler.Routes(r)
	})

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		logger.Printf("server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatalf("forced shutdown: %v", err)
	}
	logger.Println("server stopped")
}
