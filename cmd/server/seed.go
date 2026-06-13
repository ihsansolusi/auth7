package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/ihsansolusi/auth7/pkg/config"
	"github.com/ihsansolusi/lib7-service-go/logging"
)

var seedCfgFile string

func seedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Apply or rollback environment-specific seed data",
	}
	cmd.PersistentFlags().StringVar(&seedCfgFile, "config", "configs/config.yaml", "path to config file")
	cmd.AddCommand(seedUpCmd(), seedDownCmd())
	return cmd
}

func seedUpCmd() *cobra.Command {
	var profile string
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Apply seed data for the given profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSeedDir("up", profile, seedCfgFile)
		},
	}
	// --profile flag, fallback to SEED_PROFILE env var, default "demo"
	defaultProfile := os.Getenv("SEED_PROFILE")
	if defaultProfile == "" {
		defaultProfile = "demo"
	}
	cmd.Flags().StringVarP(&profile, "profile", "p", defaultProfile, "seed profile: demo, prod")
	return cmd
}

func seedDownCmd() *cobra.Command {
	var profile string
	cmd := &cobra.Command{
		Use:   "down",
		Short: "Rollback seed data for the given profile (reverse order)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSeedDir("down", profile, seedCfgFile)
		},
	}
	defaultProfile := os.Getenv("SEED_PROFILE")
	if defaultProfile == "" {
		defaultProfile = "demo"
	}
	cmd.Flags().StringVarP(&profile, "profile", "p", defaultProfile, "seed profile: demo, prod")
	return cmd
}

func runSeedDir(direction, profile, cfgPath string) error {
	cfg, err := config.Load(cfgPath)
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

	dir := filepath.Join("migrations-seed", profile)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("seed profile %q not found (expected directory: %s)", profile, dir)
	}

	pattern := filepath.Join(dir, "*."+direction+".sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob seed files: %w", err)
	}
	if len(files) == 0 {
		logger.Info().Str("profile", profile).Str("direction", direction).Msg("no seed files found")
		return nil
	}

	sort.Strings(files)
	if direction == "down" {
		for i, j := 0, len(files)-1; i < j; i, j = i+1, j-1 {
			files[i], files[j] = files[j], files[i]
		}
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, cfg.Database.Primary.DSN)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer conn.Close(ctx)

	for _, f := range files {
		sql, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}
		if _, err := conn.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("execute %s: %w", f, err)
		}
		logger.Info().Str("file", filepath.Base(f)).Msg("seed applied")
	}

	logger.Info().
		Str("profile", profile).
		Str("direction", direction).
		Int("files", len(files)).
		Msg("seed complete")
	return nil
}
