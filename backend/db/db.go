package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/sawdustofmind/bitcoin-wallet/backend/config"

	_ "github.com/lib/pq"
)

func Connect(cfg config.DBConfig) (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name)

	var db *sql.DB
	var err error

	// Retry connection loop
	for i := 0; i < 30; i++ {
		db, err = sql.Open("postgres", connStr)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		log.Printf("Failed to connect to database: %v. Retrying in 2s...", err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return nil, fmt.Errorf("could not connect to database after retries: %v", err)
	}

	if err := runMigrations(db); err != nil {
		return nil, err
	}

	return db, nil
}

func runMigrations(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS wallet_state (
		id SERIAL PRIMARY KEY,
		derivation_index INT NOT NULL DEFAULT 0
	);
	INSERT INTO wallet_state (id, derivation_index)
	SELECT 1, 0
	WHERE NOT EXISTS (SELECT 1 FROM wallet_state WHERE id = 1);
	`
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("migration failed: %v", err)
	}
	return nil
}
