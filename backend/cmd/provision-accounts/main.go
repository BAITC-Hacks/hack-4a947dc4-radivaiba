package main

import (
	"bufio"
	"careerquest/internal/store"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	if f, err := os.Open(".env"); err == nil {
		defer f.Close()
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			k, v, ok := strings.Cut(strings.TrimSpace(sc.Text()), "=")
			if ok && !strings.HasPrefix(k, "#") {
				if _, set := os.LookupEnv(k); !set {
					os.Setenv(k, strings.Trim(v, "\"'"))
				}
			}
		}
	}
	output := flag.String("output", "data/credentials/accounts.json", "Private credentials file")
	employee := flag.String("employee", "", "Only provision one employee")
	hr := flag.String("hr-login", "hr", "HR login (empty skips)")
	hrEmployee := flag.String("hr-employee", "", "Optional employee linked to HR")
	flag.Parse()
	if os.Getenv("DATABASE_URL") == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	seedDir := os.Getenv("DATA_DIR")
	if seedDir == "" {
		seedDir = "data/seed"
	}
	data, err := store.OpenPostgres(ctx, os.Getenv("DATABASE_URL"), seedDir)
	if err != nil {
		return err
	}
	defer data.Close()
	// Refuse an inaccessible destination before creating any passwords.
	if err = os.MkdirAll(filepath.Dir(*output), 0700); err != nil {
		return err
	}
	var saved struct {
		CreatedAt string             `json:"created_at"`
		Accounts  []store.Credential `json:"accounts"`
	}
	raw, err := os.ReadFile(*output)
	if err == nil {
		if err = json.Unmarshal(raw, &saved); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(*output), ".accounts-")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(0600); err != nil {
		f.Close()
		return err
	}
	created, err := data.ProvisionAccounts(*employee, *hr, *hrEmployee)
	if err != nil {
		f.Close()
		return err
	}
	saved.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	saved.Accounts = append(saved.Accounts, created...)
	if err = json.NewEncoder(f).Encode(saved); err != nil {
		f.Close()
		return fmt.Errorf("credentials were created but export failed: %w", err)
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(name, *output); err != nil {
		return err
	}
	fmt.Printf("Created %d accounts. Credentials saved privately in %s; existing passwords unchanged. Restart a running server after provisioning.\n", len(created), *output)
	return nil
}
