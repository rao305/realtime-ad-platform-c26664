package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"adplatform/platform/adserver/config"
	"adplatform/platform/adserver/decision"
	"adplatform/platform/adserver/events"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}

	rdb := decision.NewRedisClient(cfg.RedisAddr)
	view := decision.NewCampaignView(rdb)
	engine := decision.NewEngine(view, cfg.FreqCapPerHour, log)
	emitter := events.NewEmitter(cfg.KafkaBrokers, cfg.KafkaTopic, log)
	defer func() { _ = emitter.Close() }()

	ready := func(ctx context.Context) bool {
		if !cfg.ReadyRequireRedis {
			return true
		}
		cctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()
		return view.Ping(cctx) == nil
	}

	h := decision.NewHandler(engine, emitter, cfg.DecisionBudget, view, log, ready)

	mux := http.NewServeMux()
	mux.HandleFunc("/serve", h.ServeAd)
	mux.HandleFunc("/click", h.TrackClick)
	mux.HandleFunc("/healthz", h.Healthz)
	mux.HandleFunc("/readyz", h.Readyz)
	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       2 * time.Second,
		WriteTimeout:      2 * time.Second,
	}

	go func() {
		log.Info("ad server listening", "addr", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	_ = rdb.Close()
	log.Info("shutdown complete")
}
