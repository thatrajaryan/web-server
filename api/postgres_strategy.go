package api

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type PostgresStrategy struct {
	db *sql.DB
}

func (p *PostgresStrategy) Connect() error {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: No .env file found, relying on environment variables")
	}

	host := os.Getenv("hostname")
	port := os.Getenv("port")
	password := os.Getenv("POSGRES_PASSWORD")
	user := "postgres"   // Default for Postgres Docker image
	dbname := "postgres" // Default for Postgres Docker image

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return fmt.Errorf("error opening database: %v", err)
	}

	err = db.Ping()
	if err != nil {
		return fmt.Errorf("error connecting to database: %v", err)
	}

	p.db = db
	log.Printf("Successfully connected to Postgres at %s:%s", host, port)
	return nil
}

func (p *PostgresStrategy) Disconnect() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

func (p *PostgresStrategy) Query(query string, args ...interface{}) (interface{}, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	return p.db.Query(query, args...)
}
