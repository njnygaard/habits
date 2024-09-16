package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var db *sql.DB

func main() {
	// Create the root command
	rootCmd := &cobra.Command{
		Use:   "myapp",
		Short: "MyApp is a sample application",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Welcome to MyApp! Use --help to see available commands.")
		},
	}

	// Initialize Viper
	if err := initConfig(); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}

	// Initialize SQLite database
	if err := initDB(); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {

		}
	}(db)

	// Add a sample subcommand
	rootCmd.AddCommand(&cobra.Command{
		Use:   "create",
		Short: "Create a sample table and insert data",
		Run: func(cmd *cobra.Command, args []string) {
			if err := createSampleData(); err != nil {
				_, err := fmt.Fprintf(os.Stderr, "Error creating sample data: %v\n", err)
				if err != nil {
					return
				}
				os.Exit(1)
			}
			fmt.Println("Sample data created successfully!")
		},
	})

	// Add a flag for the hello command
	rootCmd.Commands()[0].Flags().String("name", "world", "Name to greet")

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() error {
	// Set default config file name and path
	viper.SetConfigName("config") // name of config file (without extension)
	viper.AddConfigPath(".")      // optionally look for config in the working directory
	viper.AutomaticEnv()          // read environment variables

	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	return nil
}

// initDB initializes the SQLite database connection.
func initDB() error {
	var err error
	dbPath := viper.GetString("database.path")
	if dbPath == "" {
		dbPath = "database.db" // default database path
	}

	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("error opening database: %w", err)
	}

	// Create a table if it does not exist
	createTableSQL := `CREATE TABLE IF NOT EXISTS sample (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("error creating table: %w", err)
	}

	return nil
}

// createSampleData inserts sample data into the database.
func createSampleData() error {
	_, err := db.Exec("INSERT INTO sample (name) VALUES (?)", "example")
	if err != nil {
		return fmt.Errorf("error inserting sample data: %w", err)
	}
	return nil
}
