package main

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/spf13/cobra"
	"github.com/rs/zerolog"

	"github.com/ihsansolusi/auth7/pkg/config"
	"github.com/ihsansolusi/lib7-service-go/logging"
)

var migrateCfgFile string

func migrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
	}
	cmd.PersistentFlags().StringVar(&migrateCfgFile, "config", "configs/config.yaml", "path to config file")

	cmd.AddCommand(upCmd(), downCmd(), migrateVersionCmd(), forceCmd())
	return cmd
}

func upCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Apply all pending migrations",
		RunE:  runMigrateUp,
	}
}

func runMigrateUp(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(migrateCfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logLevel, err := zerolog.ParseLevel(cfg.Logging.Level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}
	logger := logging.NewLogger(logging.Options{
		Level:  logLevel,
		Pretty: cfg.Logging.Pretty,
	})

	m, err := newMigrator(cfg.Database.Primary.DSN, logger)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}

	logger.Info().Msg("migrations applied successfully")
	return nil
}

func downCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Rollback the last migration",
		RunE:  runMigrateDown,
	}
}

func runMigrateDown(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(migrateCfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logLevel, err := zerolog.ParseLevel(cfg.Logging.Level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}
	logger := logging.NewLogger(logging.Options{
		Level:  logLevel,
		Pretty: cfg.Logging.Pretty,
	})

	m, err := newMigrator(cfg.Database.Primary.DSN, logger)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Steps(-1); err != nil {
		return fmt.Errorf("migrate down: %w", err)
	}

	logger.Info().Msg("migrations rolled back")
	return nil
}

func migrateVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print current migration version",
		RunE:  runMigrateVersion,
	}
}

func runMigrateVersion(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(migrateCfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logLevel, err := zerolog.ParseLevel(cfg.Logging.Level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}
	logger := logging.NewLogger(logging.Options{
		Level:  logLevel,
		Pretty: cfg.Logging.Pretty,
	})

	m, err := newMigrator(cfg.Database.Primary.DSN, logger)
	if err != nil {
		return err
	}
	defer m.Close()

	version, dirty, err := m.Version()
	if err != nil {
		return fmt.Errorf("get version: %w", err)
	}

	logger.Info().
		Int64("version", int64(version)).
		Bool("dirty", dirty).
		Msg("current migration version")

	return nil
}

func forceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "force [version]",
		Short: "Set migration version (use with caution)",
		Args:  cobra.ExactArgs(1),
		RunE:  runMigrateForce,
	}
}

func runMigrateForce(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(migrateCfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logLevel, err := zerolog.ParseLevel(cfg.Logging.Level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}
	logger := logging.NewLogger(logging.Options{
		Level:  logLevel,
		Pretty: cfg.Logging.Pretty,
	})

	m, err := newMigrator(cfg.Database.Primary.DSN, logger)
	if err != nil {
		return err
	}
	defer m.Close()

	var version uint
	if _, err := fmt.Sscanf(args[0], "%d", &version); err != nil {
		return fmt.Errorf("invalid version: %w", err)
	}

	if err := m.Force(int(version)); err != nil {
		return fmt.Errorf("force version: %w", err)
	}

	logger.Info().Uint("version", version).Msg("migration version forced")
	return nil
}

func newMigrator(dsn string, logger zerolog.Logger) (*migrate.Migrate, error) {
	m, err := migrate.New(
		"file://migrations",
		dsn,
	)
	if err != nil {
		return nil, fmt.Errorf("create migrator: %w", err)
	}

	return m, nil
}
