package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/thatrajaryan/web-server/common"
)

type DatabaseBlock struct {
	Engine       string
	Host         string
	Port         int
	Username     string
	Password     string
	DatabaseName string
	StorageGB    int
	
	// Live clients
	sqlDB   *sql.DB
	mongoDB *mongo.Client
}

func (b *DatabaseBlock) Create(config map[string]interface{}) error {
	fmt.Println("[Database] Creating new database instance...")
	return b.Update(config)
}

func (b *DatabaseBlock) Connect(target common.Block) error {
	fmt.Printf("[Database] Instance (%s) establishing connection to target block\n", b.Engine)
	return nil
}

func (b *DatabaseBlock) Update(config map[string]interface{}) error {
	// 1. Close existing connections
	b.cleanup()

	// 2. Parse new configuration
	if val, ok := config["engine"].(string); ok {
		b.Engine = val
	}
	if val, ok := config["host"].(string); ok {
		b.Host = val
	}
	if val, ok := config["port"].(float64); ok {
		b.Port = int(val)
	}
	if val, ok := config["username"].(string); ok {
		b.Username = val
	}
	if val, ok := config["password"].(string); ok {
		b.Password = val
	}
	if val, ok := config["database_name"].(string); ok {
		b.DatabaseName = val
	}
	if val, ok := config["storage_gb"].(float64); ok {
		b.StorageGB = int(val)
	}

	fmt.Printf("[Database] Connecting to %s at %s:%d/%s...\n", b.Engine, b.Host, b.Port, b.DatabaseName)

	// 3. Initialize real connection
	var err error
	switch b.Engine {
	case "postgres":
		err = b.initPostgres()
	case "mysql":
		err = b.initMySQL()
	case "mongodb":
		err = b.initMongoDB()
	default:
		return fmt.Errorf("unsupported database engine: %s", b.Engine)
	}

	if err != nil {
		fmt.Printf("[Database] [ERROR] Connection failed: %v\n", err)
		return err
	}

	fmt.Printf("[Database] Successfully connected to %s\n", b.Engine)
	return nil
}

func (b *DatabaseBlock) initPostgres() error {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", 
		b.Host, b.Port, b.Username, b.Password, b.DatabaseName)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return err
	}
	
	b.sqlDB = db
	return nil
}

func (b *DatabaseBlock) initMySQL() error {
	connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", 
		b.Username, b.Password, b.Host, b.Port, b.DatabaseName)
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return err
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return err
	}
	
	b.sqlDB = db
	return nil
}

func (b *DatabaseBlock) initMongoDB() error {
	uri := fmt.Sprintf("mongodb://%s:%s@%s:%d", b.Username, b.Password, b.Host, b.Port)
	if b.Username == "" || b.Password == "" {
		uri = fmt.Sprintf("mongodb://%s:%d", b.Host, b.Port)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}
	
	if err := client.Ping(ctx, nil); err != nil {
		client.Disconnect(ctx)
		return err
	}
	
	b.mongoDB = client
	return nil
}

func (b *DatabaseBlock) cleanup() {
	if b.sqlDB != nil {
		b.sqlDB.Close()
		b.sqlDB = nil
	}
	if b.mongoDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		b.mongoDB.Disconnect(ctx)
		b.mongoDB = nil
	}
}

func (b *DatabaseBlock) Delete() error {
	b.cleanup()
	fmt.Printf("[Database] Instance (%s) deleted and connections closed\n", b.Engine)
	return nil
}
