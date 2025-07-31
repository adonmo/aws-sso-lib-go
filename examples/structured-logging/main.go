package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/adonmo/aws-sso-lib-go/awsssolib"
)

func main() {
	// Create a structured logger with JSON output
	jsonLogger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug, // Enable debug logging to see all details
	}))

	// Create configuration with custom logger
	config := awsssolib.NewConfig(jsonLogger, slog.LevelDebug)

	// Example configuration - replace with your values
	startURL := os.Getenv("AWS_DEFAULT_SSO_START_URL")
	ssoRegion := os.Getenv("AWS_DEFAULT_SSO_REGION")

	if startURL == "" || ssoRegion == "" {
		slog.Error("Missing required environment variables")
		os.Exit(1)
	}

	ctx := context.Background()

	// Login with structured logging
	_, err := awsssolib.Login(ctx, awsssolib.LoginInput{
		StartURL:  startURL,
		SSORegion: ssoRegion,
		Config:    config, // Pass the config with custom logger
	})
	if err != nil {
		slog.Error("Login failed", slog.Any("error", err))
		os.Exit(1)
	}

	// List accounts with structured logging
	accounts, err := awsssolib.ListAvailableAccounts(ctx, awsssolib.ListAccountsInput{
		StartURL:  startURL,
		SSORegion: ssoRegion,
		Config:    config, // Pass the config with custom logger
	})
	if err != nil {
		slog.Error("Failed to list accounts", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("Available accounts", slog.Int("count", len(accounts)))
	for _, account := range accounts {
		slog.Info("Account found", 
			slog.String("account_id", account.AccountID),
			slog.String("account_name", account.AccountName),
			slog.String("email", account.EmailAddress))
	}

	// You can also use the default config (INFO level logging to stderr)
	defaultConfig := awsssolib.DefaultConfig()
	
	slog.Info("Using default config for role listing")
	roles, err := awsssolib.ListAvailableRoles(ctx, awsssolib.ListRolesInput{
		StartURL:  startURL,
		SSORegion: ssoRegion,
		Config:    defaultConfig,
	})
	if err != nil {
		slog.Error("Failed to list roles", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("Available roles", slog.Int("count", len(roles)))
}