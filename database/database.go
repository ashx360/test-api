package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

func InitDB(connectionString string) (*sql.DB, error) {
	if connectionString == "" {
		return nil, fmt.Errorf("DATABASE_URL is empty")
	}

	// Force SSL for Railway
	if !strings.Contains(connectionString, "sslmode=") {
		if strings.Contains(connectionString, "?") {
			connectionString += "&sslmode=require"
		} else {
			connectionString += "?sslmode=require"
		}
	}

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	log.Println("Database connected successfully")
	return db, nil
}
