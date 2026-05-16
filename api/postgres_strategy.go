package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/thatrajaryan/web-server/api/models"
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

func (p *PostgresStrategy) Exec(query string, args ...interface{}) (sql.Result, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	return p.db.Exec(query, args...)
}

func (p *PostgresStrategy) Begin() (*sql.Tx, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	return p.db.Begin()
}

func (p *PostgresStrategy) Migrate() error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}

	schema, err := os.ReadFile("api/models/schema.sql")
	if err != nil {
		return fmt.Errorf("error reading schema file: %v", err)
	}

	_, err = p.db.Exec(string(schema))
	if err != nil {
		return fmt.Errorf("error executing schema: %v", err)
	}

	log.Println("Database migration completed successfully")
	return nil
}

func (p *PostgresStrategy) GetNodesByProject(projectID string) ([]models.Node, error) {
	rows, err := p.db.Query("SELECT id, project_id, type, config, created_at, updated_at FROM nodes WHERE project_id = $1", projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []models.Node
	for rows.Next() {
		var n models.Node
		var configStr string
		if err := rows.Scan(&n.ID, &n.ProjectID, &n.Type, &configStr, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(configStr), &n.Config)
		nodes = append(nodes, n)
	}
	return nodes, nil
}

func (p *PostgresStrategy) GetConnectionsByProject(projectID string) ([]models.Connection, error) {
	rows, err := p.db.Query("SELECT id, project_id, from_node_id, to_node_id, hook_code, created_at FROM connections WHERE project_id = $1", projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conns []models.Connection
	for rows.Next() {
		var c models.Connection
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.FromNodeID, &c.ToNodeID, &c.HookCode, &c.CreatedAt); err != nil {
			return nil, err
		}
		conns = append(conns, c)
	}
	return conns, nil
}
func (p *PostgresStrategy) GetProject(id string) (*models.Project, error) {
	row := p.db.QueryRow("SELECT id, name, description, created_at, updated_at FROM projects WHERE id = $1", id)
	var proj models.Project
	err := row.Scan(&proj.ID, &proj.Name, &proj.Description, &proj.CreatedAt, &proj.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &proj, nil
}
