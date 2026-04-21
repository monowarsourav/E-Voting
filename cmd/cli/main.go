// cmd/cli/main.go

package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/database"
	_ "github.com/mattn/go-sqlite3"
)

const (
	dbPath = "./data/covertvote.db"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "migrate":
		runMigrations()
	case "genkeys":
		generateKeys()
	case "createdb":
		createDatabase()
	case "resetdb":
		resetDatabase()
	case "stats":
		showStats()
	case "help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("CovertVote CLI Tool")
	fmt.Println("\nUsage:")
	fmt.Println("  go run cmd/cli/main.go <command>")
	fmt.Println("\nCommands:")
	fmt.Println("  migrate   - Run database migrations")
	fmt.Println("  genkeys   - Generate cryptographic keys")
	fmt.Println("  createdb  - Create database")
	fmt.Println("  resetdb   - Reset database (WARNING: deletes all data)")
	fmt.Println("  stats     - Show database statistics")
	fmt.Println("  help      - Show this help message")
}

func runMigrations() {
	fmt.Println("Running database migrations...")

	// Ensure data directory exists
	dataDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Printf("Failed to create data directory: %v\n", err)
		os.Exit(1)
	}

	// Open database using wrapper
	db, err := database.New(dbPath)
	if err != nil {
		fmt.Printf("Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Run migrations
	migrationsDir := "./migrations"
	if err := db.RunMigrations(migrationsDir); err != nil {
		fmt.Printf("Migration failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Migrations completed successfully!")
}

func generateKeys() {
	fmt.Println("Generating cryptographic keys...")

	// Generate Paillier key pair (2048-bit)
	fmt.Println("Generating Paillier key pair (2048-bit)...")
	paillierSK, err := crypto.GeneratePaillierKeyPair(2048)
	if err != nil {
		fmt.Printf("Failed to generate Paillier keys: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Paillier public key (N): %s\n", paillierSK.PublicKey.N.String()[:64]+"...")

	// Generate Pedersen parameters (512-bit)
	fmt.Println("Generating Pedersen parameters (512-bit)...")
	pedersenParams, err := crypto.GeneratePedersenParams(512)
	if err != nil {
		fmt.Printf("Failed to generate Pedersen params: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Pedersen P: %s\n", pedersenParams.P.String()[:64]+"...")

	// Generate Ring signature parameters (512-bit)
	fmt.Println("Generating Ring signature parameters (512-bit)...")
	ringParams, err := crypto.GenerateRingParams(512)
	if err != nil {
		fmt.Printf("Failed to generate Ring params: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Ring P: %s\n", ringParams.P.String()[:64]+"...")

	fmt.Println("\nKey generation completed successfully!")
	fmt.Println("Note: In production, save these keys securely!")
}

func createDatabase() {
	fmt.Println("Creating database...")

	// Ensure data directory exists
	dataDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Printf("Failed to create data directory: %v\n", err)
		os.Exit(1)
	}

	// Create database file
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		fmt.Printf("Failed to create database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Database created successfully at: %s\n", dbPath)
	fmt.Println("Run 'migrate' command to create tables.")
}

func resetDatabase() {
	fmt.Println("WARNING: This will delete all data!")
	fmt.Print("Are you sure? (yes/no): ")

	var confirmation string
	fmt.Scanln(&confirmation)

	if confirmation != "yes" {
		fmt.Println("Operation cancelled.")
		return
	}

	// Delete database file
	if err := os.Remove(dbPath); err != nil {
		if !os.IsNotExist(err) {
			fmt.Printf("Failed to delete database: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("Database deleted.")

	// Recreate database
	createDatabase()

	// Run migrations
	runMigrations()

	fmt.Println("Database reset complete!")
}

func showStats() {
	fmt.Println("Fetching database statistics...")

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		fmt.Printf("Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Count elections
	var electionCount int
	err = db.QueryRow("SELECT COUNT(*) FROM elections").Scan(&electionCount)
	if err != nil {
		fmt.Printf("Failed to count elections: %v\n", err)
		electionCount = 0
	}

	// Count voters
	var voterCount int
	err = db.QueryRow("SELECT COUNT(*) FROM voters").Scan(&voterCount)
	if err != nil {
		fmt.Printf("Failed to count voters: %v\n", err)
		voterCount = 0
	}

	// Count ballots
	var ballotCount int
	err = db.QueryRow("SELECT COUNT(*) FROM ballots").Scan(&ballotCount)
	if err != nil {
		fmt.Printf("Failed to count ballots: %v\n", err)
		ballotCount = 0
	}

	// Count credentials
	var credentialCount int
	err = db.QueryRow("SELECT COUNT(*) FROM credentials").Scan(&credentialCount)
	if err != nil {
		fmt.Printf("Failed to count credentials: %v\n", err)
		credentialCount = 0
	}

	// Print statistics
	fmt.Println("\n=== Database Statistics ===")
	fmt.Printf("Total Elections:   %d\n", electionCount)
	fmt.Printf("Total Voters:      %d\n", voterCount)
	fmt.Printf("Total Ballots:     %d\n", ballotCount)
	fmt.Printf("Total Credentials: %d\n", credentialCount)

	// Get election status breakdown
	rows, err := db.Query("SELECT status, COUNT(*) FROM elections GROUP BY status")
	if err == nil {
		defer rows.Close()
		fmt.Println("\nElections by Status:")
		for rows.Next() {
			var status string
			var count int
			if err := rows.Scan(&status, &count); err == nil {
				fmt.Printf("  %s: %d\n", status, count)
			}
		}
	}

	// Get voter participation
	var votedCount int
	err = db.QueryRow("SELECT COUNT(*) FROM voters WHERE has_voted = 1").Scan(&votedCount)
	if err == nil && voterCount > 0 {
		percentage := float64(votedCount) / float64(voterCount) * 100
		fmt.Printf("\nVoter Participation: %d/%d (%.2f%%)\n", votedCount, voterCount, percentage)
	}
}
