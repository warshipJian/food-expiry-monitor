package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/james/food-expiry-monitor/internal/app"
	"github.com/james/food-expiry-monitor/internal/store"
	_ "modernc.org/sqlite"
)

func main() {
	cfg := app.LoadConfig()
	db, err := sql.Open("sqlite", cfg.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := store.Migrate(db); err != nil {
		log.Fatal(err)
	}

	service := app.NewService(db, cfg)
	go service.RunReminderScheduler(context.Background())

	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	server := &http.Server{Addr: cfg.HTTPAddr, Handler: service.Router()}
	go func() {
		log.Printf("food-expiry API listening on %s", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
