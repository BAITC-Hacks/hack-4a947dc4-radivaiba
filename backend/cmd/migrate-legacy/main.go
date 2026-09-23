package main

import (
	"careerquest/internal/store"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	state := flag.String("state", "", "path to the backed-up complete legacy state.json")
	dryRun := flag.Bool("dry-run", false, "validate snapshot and empty target without writing")
	apply := flag.Bool("apply", false, "import the snapshot into an empty PostgreSQL database")
	flag.Parse()
	if *state == "" || *dryRun == *apply || os.Getenv("DATABASE_URL") == "" {
		fmt.Fprintln(os.Stderr, "Usage: set DATABASE_URL, then migrate-legacy --state <snapshot.json> (--dry-run | --apply)")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	report, err := store.MigrateLegacy(ctx, os.Getenv("DATABASE_URL"), *state, *apply)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err = json.NewEncoder(os.Stdout).Encode(report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
