package main

import (
	"bufio"
	"careerquest/internal/auth"
	"careerquest/internal/engine"
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
)

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
	employeePassword, hrPassword := os.Getenv("DEMO_EMPLOYEE_PASSWORD"), os.Getenv("DEMO_HR_PASSWORD")
	if employeePassword == "" || hrPassword == "" || employeePassword == hrPassword {
		log.Fatal("Set separate DEMO_EMPLOYEE_PASSWORD and DEMO_HR_PASSWORD in .env (npm run setup generates them).")
	}
	employeeID := env("DEMO_EMPLOYEE_ID", "E0001")
	if _, ok := engine.FindEmployee(data.Snapshot(), employeeID); !ok {
		log.Fatal("DEMO_EMPLOYEE_ID is missing in dataset")
	}
	timeout, _ := strconv.Atoi(env("LLM_TIMEOUT_MS", "8000"))
	api := &httpapi.Server{Store: data, Auth: auth.New(employeeID, employeePassword, hrPassword), LLM: &llm.Client{BaseURL: env("LLM_BASE_URL", "https://api.openai.com/v1"), Model: os.Getenv("LLM_MODEL"), APIKey: os.Getenv("LLM_API_KEY"), Timeout: time.Duration(timeout) * time.Millisecond}, WebDir: env("WEB_DIR", "frontend/dist"), DevOrigin: os.Getenv("DEV_ORIGIN")}
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
