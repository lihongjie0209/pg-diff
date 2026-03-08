package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/pg-diff/internal/adapter/flyway"
	"github.com/user/pg-diff/internal/adapter/generator"
	"github.com/user/pg-diff/internal/adapter/postgres"
	"github.com/user/pg-diff/internal/usecase"
)

var (
	sourceConn  string
	targetConn  string
	schemaName  string
	flywayDir   string
	objectTypes string
	dryRun      bool
)

var rootCmd = &cobra.Command{
	Use:   "pg-diff",
	Short: "PostgreSQL schema diff tool",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Initialize Adapters
		sourceRepo, err := postgres.NewPostgresRepository(sourceConn)
		if err != nil {
			return fmt.Errorf("failed to connect to source: %w", err)
		}
		targetRepo, err := postgres.NewPostgresRepository(targetConn)
		if err != nil {
			return fmt.Errorf("failed to connect to target: %w", err)
		}

		gen := generator.NewSQLGenerator()
		diffUC := usecase.NewDiffService(gen)

		// 2. Fetch Schemas
		sourceSchema, err := sourceRepo.GetSchema(schemaName)
		if err != nil {
			return fmt.Errorf("failed to fetch source schema: %w", err)
		}
		targetSchema, err := targetRepo.GetSchema(schemaName)
		if err != nil {
			return fmt.Errorf("failed to fetch target schema: %w", err)
		}

		// Apply selective filtering based on object types if provided
		if objectTypes != "" {
			typesList := strings.Split(objectTypes, ",")
			sourceSchema = usecase.FilterSchema(sourceSchema, typesList)
			targetSchema = usecase.FilterSchema(targetSchema, typesList)

			// Optional: We can merge their Requirement Sets together across both Source/Target
			// For simplicity and correctness with the existing BFS, we'll synchronize dependencies
			// accurately enough by checking each graph. Diffing will compare the trimmed subsets.
		}

		// 3. Compare and Output
		actions, err := diffUC.Compare(sourceSchema, targetSchema)
		if err != nil {
			return fmt.Errorf("diff failed: %w", err)
		}

		if len(actions) == 0 {
			fmt.Println("No schema differences found.")
			return nil
		}

		if flywayDir != "" {
			return writeFlywayFile(actions)
		}

		for _, action := range actions {
			fmt.Println("-- Action:", action.Type, action.Object, action.Name)
			fmt.Println(action.SQL)
			fmt.Println()
		}

		return nil
	},
}

func writeFlywayFile(actions []usecase.MigrationAction) error {
	nextVer, err := flyway.GetNextVersion(flywayDir, true)
	if err != nil {
		return fmt.Errorf("failed to get next flyway version: %w", err)
	}

	fileName := fmt.Sprintf("V%s__pg_diff_migration.sql", nextVer)
	filePath := filepath.Join(flywayDir, fileName)

	var sb strings.Builder
	for _, action := range actions {
		sb.WriteString(fmt.Sprintf("-- Action: %s %s %s\n", action.Type, action.Object, action.Name))
		sb.WriteString(action.SQL + "\n\n")
	}

	if dryRun {
		fmt.Printf("[DRY-RUN] Would create file: %s\n", filePath)
		fmt.Println("------------- SCRIPT PREVIEW -------------")
		fmt.Println(sb.String())
		fmt.Println("------------------------------------------")
		return nil
	}

	err = os.MkdirAll(flywayDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create flyway directory: %w", err)
	}

	err = os.WriteFile(filePath, []byte(sb.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write flyway file: %w", err)
	}

	fmt.Printf("Successfully generated flyway file: %s\n", filePath)
	return nil
}

func Execute() {
	rootCmd.Flags().StringVar(&sourceConn, "source", "", "Source connection string (Postgres URL)")
	rootCmd.Flags().StringVar(&targetConn, "target", "", "Target connection string (Postgres URL)")
	rootCmd.Flags().StringVar(&schemaName, "schema", "public", "Schema name to diff")
	rootCmd.Flags().StringVar(&flywayDir, "flyway-dir", "", "Directory to output Flyway migration scripts. Output to console if empty.")
	rootCmd.Flags().StringVarP(&objectTypes, "object-types", "o", "", "Comma-separated list of object types to sync (e.g. 'table,view,function'). Dependencies are pulled automatically.")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview the script without writing a file (requires --flyway-dir)")

	rootCmd.MarkFlagRequired("source")
	rootCmd.MarkFlagRequired("target")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
