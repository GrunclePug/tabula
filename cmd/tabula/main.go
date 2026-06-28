package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/grunclepug/tabula/internal/database"
	"github.com/grunclepug/tabula/internal/server"
	"github.com/joho/godotenv"
)

var version = "dev"

func main() {
	log.Printf("Starting Tabula v%s", version)

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	docsRoot := os.Getenv("TABULA_DOCS_ROOT")
	if docsRoot == "" {
		log.Fatal("TABULA_DOCS_ROOT environment variable is required")
	}

	// Validate docsRoot exists
	if stat, err := os.Stat(docsRoot); err != nil || !stat.IsDir() {
		log.Fatalf("TABULA_DOCS_ROOT '%s' is not a valid directory", docsRoot)
	}

	dataDir := os.Getenv("TABULA_DATA_DIR")
	if dataDir == "" {
		dataDir = "."
	}

	// Ensure dataDir exists
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Fatalf("Failed to create TABULA_DATA_DIR: %v", err)
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = filepath.Join(dataDir, "tabula.db")
	}

	db, err := database.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	srv := server.New(db, docsRoot, dataDir, version)
	log.Printf("Server listening on %s", addr)
	if err := srv.Start(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
