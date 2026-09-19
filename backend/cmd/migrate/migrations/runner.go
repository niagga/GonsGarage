// File: backend/cmd/migrate/migrations/runner.go
package migrations

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func createMigrationsTable(db *sql.DB) error {
	query := `
    CREATE TABLE IF NOT EXISTS schema_migrations (
        version VARCHAR(255) PRIMARY KEY,
        applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );`

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}
	return nil
}

// Run applies pending migrations from migrationsDir.
func Run(db *sql.DB, migrationsDir string) error {
	// Create migrations table if it doesn't exist
	if err := createMigrationsTable(db); err != nil {
		return err
	}

	// Get applied migrations
	appliedMigrations := getAppliedMigrations(db)

	// Get migration files
	migrationFiles, err := getMigrationFiles(migrationsDir)
	if err != nil {
		return err
	}

	// Run pending migrations
	for _, file := range migrationFiles {
		if !strings.HasSuffix(file, ".up.sql") {
			continue
		}

		version := strings.TrimSuffix(file, ".up.sql")

		if _, exists := appliedMigrations[version]; exists {
			fmt.Printf("⏭️  Skipping already applied migration: %s\n", version)
			continue
		}

		fmt.Printf("🔄 Running migration: %s\n", version)

		if err := runMigration(db, migrationsDir, file, version); err != nil {
			return fmt.Errorf("failed to run migration %s: %w", version, err)
		}

		fmt.Printf("✅ Completed migration: %s\n", version)
	}

	return nil
}

func getAppliedMigrations(db *sql.DB) map[string]bool {
	applied := make(map[string]bool)

	rows, err := db.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return applied
	}
	defer rows.Close()

	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err == nil {
			applied[version] = true
		}
	}

	return applied
}

func getMigrationFiles(migrationsDir string) ([]string, error) {
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrationFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			migrationFiles = append(migrationFiles, file.Name())
		}
	}

	sort.Strings(migrationFiles)
	return migrationFiles, nil
}

func runMigration(db *sql.DB, migrationsDir, filename, version string) error {
	// Read migration file
	content, err := os.ReadFile(filepath.Join(migrationsDir, filename))
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration
	if _, err := tx.Exec(string(content)); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	// Record migration as applied
	if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	return nil
}
