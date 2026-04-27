package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Version = "dev"

var cfgFile string

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "auth7",
		Short: "auth7 — Identity & Access Management Platform",
		Long:  "Auth7 is the IAM platform for Core7 ecosystem.",
	}

	root.PersistentFlags().StringVar(&cfgFile, "config", "configs/config.yaml", "path to the config YAML file")

	root.AddCommand(
		startCmd(),
		migrateCmd(),
		versionCmd(),
	)

	return root
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the binary version and exit",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("auth7 version %s\n", Version)
		},
	}
}
