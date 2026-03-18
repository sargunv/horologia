package main

import (
	"fmt"
	"os"

	"github.com/sargunv/tend/server/internal/database"
)

func main() {
	dbPath := os.Getenv("TEND_DB")
	if dbPath == "" {
		dbPath = "tend.db"
	}

	db, err := database.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	fmt.Println("tend-server: database ready")
}
