package main

import (
	"fmt"
	"os"

	"github.com/adonmo/aws-sso-lib-go/cmd/aws-sso-util/commands"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:   "aws-sso-util",
		Short: "AWS SSO utility for easier AWS IAM Identity Center usage",
		Long: `aws-sso-util makes working with AWS IAM Identity Center (formerly AWS SSO) easier.

It provides utilities for:
- Configuring AWS CLI profiles
- Logging in/out
- Listing available accounts and roles
- Running commands with specific credentials
- Opening AWS Console in browser
- Admin functions for SSO management`,
		Version: version,
	}

	// Global flags
	rootCmd.PersistentFlags().String("start-url", "", "AWS SSO start URL")
	rootCmd.PersistentFlags().String("sso-region", "", "AWS SSO region")

	// Add commands
	rootCmd.AddCommand(commands.NewConfigureCommand())
	rootCmd.AddCommand(commands.NewLoginCommand())
	rootCmd.AddCommand(commands.NewLogoutCommand())
	rootCmd.AddCommand(commands.NewRolesCommand())
	rootCmd.AddCommand(commands.NewRunAsCommand())
	rootCmd.AddCommand(commands.NewConsoleCommand())
	rootCmd.AddCommand(commands.NewCheckCommand())
	rootCmd.AddCommand(commands.NewAdminCommand())
	rootCmd.AddCommand(commands.NewCredentialProcessCommand())

	// Set version template
	rootCmd.SetVersionTemplate(`{{printf "%s\n" .Version}}`)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}