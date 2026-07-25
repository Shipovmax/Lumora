// Command migrate — тонкая обёртка над goose (библиотекой, не CLI), чтобы не
// тянуть в зависимости драйверы всех СУБД, которые поддерживает goose/cmd/goose.
package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/Shipovmax/Lumora/internal/config"
)

const migrationsDir = "migrations"

func main() {
	if len(os.Args) != 2 || (os.Args[1] != "up" && os.Args[1] != "down") {
		fmt.Fprintln(os.Stderr, "usage: migrate [up|down]")
		os.Exit(2)
	}

	if err := run(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func run(direction string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	db, err := sql.Open("pgx", cfg.Postgres.DSN())
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	switch direction {
	case "up":
		return goose.Up(db, migrationsDir)
	case "down":
		return goose.Down(db, migrationsDir)
	default:
		return fmt.Errorf("unknown direction %q", direction)
	}
}
