package main

import (
	"bufio"
	"careerquest/internal/auth"
	"careerquest/internal/httpapi"
	"careerquest/internal/llm"
	"careerquest/internal/store"
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata"
)

var buildVersion = "dev"

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
func loadEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		if _, exists := os.LookupEnv(k); !exists {
			os.Setenv(k, strings.Trim(strings.TrimSpace(v), "\"'"))
		}
	}
}
func main() {
	loadEnv(".env")
	seedDir := env("DATA_DIR", "data/seed")
	if os.Getenv("VALIDATE_ONLY") == "1" {
		d, err := store.LoadSeed(seedDir)
		if err != nil {
			log.Fatalf("Cannot validate seed files: %v", err)
		}
		log.Printf("Validated: employees=%d events=%d skills=%d history=%d", len(d.Employees), len(d.Events), len(d.Catalog.Skills), len(d.History))
		return
	}
	if os.Getenv("DATABASE_URL") == "" {
		log.Fatal("Set DATABASE_URL in .env, then start PostgreSQL (docker compose up -d postgres).")
	}
	startup, cancelStartup := context.WithTimeout(context.Background(), 30*time.Second)
	data, err := store.OpenPostgres(startup, os.Getenv("DATABASE_URL"), seedDir)
	cancelStartup()
	if err != nil {
		log.Fatalf("Cannot open PostgreSQL dataset: %v", err)
	}
	defer data.Close()
	zoneName := env("ORG_TIMEZONE", "Asia/Qyzylorda")
	_, err = time.LoadLocation(zoneName)
	if err != nil {
		log.Fatal("Invalid ORG_TIMEZONE")
	}
	if env("APP_MODE", "demo") == "live" {
		err = data.ConfigureLiveClock(zoneName, nil)
	} else {
		err = data.SetBusinessDate(env("DEMO_DATE", data.Snapshot().Catalog.Meta.AsOfDate))
	}
	if err != nil {
		log.Fatalf("Business calendar: %v", err)
	}
	if err = data.EnsureConfigs(); err != nil {
		log.Fatalf("Module configuration: %v", err)
	}
	if len(data.Snapshot().Workflow.Accounts) == 0 {
		log.Print("No accounts yet. Run npm run accounts:provision, then restart the server.")
	}
	timeout, _ := strconv.Atoi(env("LLM_TIMEOUT_MS", "8000"))
	api := &httpapi.Server{Store: data, Auth: auth.New("", "", ""), LLM: &llm.Client{BaseURL: env("LLM_BASE_URL", "https://api.openai.com/v1"), Model: os.Getenv("LLM_MODEL"), APIKey: os.Getenv("LLM_API_KEY"), Timeout: time.Duration(timeout) * time.Millisecond}, WebDir: env("WEB_DIR", "frontend/dist"), DevOrigin: os.Getenv("DEV_ORIGIN"), EvidenceDir: env("EVIDENCE_DIR", "data/evidence"), BuildVersion: env("BUILD_VERSION", buildVersion)}
	server := &http.Server{Addr: net.JoinHostPort(env("HOST", "127.0.0.1"), env("PORT", "8080")), Handler: api.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdown)
	}()
	log.Printf("Career Quest ready: http://%s · snapshot %s", server.Addr, data.Snapshot().Catalog.Meta.AsOfDate)
	if err = server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
