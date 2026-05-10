package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"emptyclassroom/internal/config"
	"emptyclassroom/internal/crawler"
	"emptyclassroom/internal/handler"
	"emptyclassroom/internal/repository"
	"emptyclassroom/internal/service"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	cfg, err := config.Load(".env")
	if err != nil {
		logger.Fatalf("load config: %v", err)
	}

	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		logger.Printf("load timezone %q failed, fallback to local: %v", cfg.Timezone, err)
		loc = time.Local
	}

	db, err := openDB(cfg.DatabaseDSN)
	if err != nil {
		logger.Fatalf("open database: %v", err)
	}
	defer db.Close()

	repo := repository.NewMySQL(db)
	startupCtx, cancelStartup := context.WithTimeout(context.Background(), 30*time.Second)
	if cfg.AutoMigrate {
		if err := repo.Migrate(startupCtx); err != nil {
			cancelStartup()
			logger.Fatalf("migrate database: %v", err)
		}
	}
	cancelStartup()

	crawlerClient, err := crawler.NewClient(cfg.BUPT)
	if err != nil {
		logger.Fatalf("create crawler: %v", err)
	}

	classroomService := service.NewClassroomService(repo, crawlerClient, loc)

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.SyncOnStart {
		go func() {
			if err := classroomService.SyncAllToday(rootCtx, []int{0, 1}); err != nil {
				logger.Printf("startup sync failed: %v", err)
			}
		}()
	}
	if cfg.EnableScheduler {
		service.StartScheduler(rootCtx, logger, classroomService, loc)
	}

	httpHandler := handler.New(classroomService, cfg)
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpHandler.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Printf("server listening on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatalf("listen: %v", err)
		}
	}()

	<-rootCtx.Done()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Printf("shutdown server: %v", err)
	}
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(15)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
