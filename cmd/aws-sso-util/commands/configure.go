package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/adonmo/aws-sso-lib-go/awsssolib"
	"github.com/spf13/cobra"
)

// NewConfigureCommand creates the configure command group
func NewConfigureCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Configure AWS CLI profiles",
		Long:  `Commands to configure AWS CLI profiles for SSO access.`,
	}

	cmd.AddCommand(newConfigureProfileCommand())
	cmd.AddCommand(newConfigurePopulateCommand())

	return cmd
}

// newConfigureProfileCommand creates the configure profile command
func newConfigureProfileCommand() *cobra.Command {
	var region string
	var outputFormat string
	var credentialProcess bool

	cmd := &cobra.Command{
		Use:   "profile <profile-name>",
		Short: "Configure a single AWS CLI profile",
		Long: `Configure a single AWS CLI profile for SSO access.

This command will interactively prompt you to select an account and role
from those available through your SSO access.

Examples:
  # Configure a profile interactively
  aws-sso-util configure profile my-profile

  # Configure with specific region
  aws-sso-util configure profile my-profile --region us-west-2

  # Add credential process support
  aws-sso-util configure profile my-profile --credential-process`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			profileName := args[0]

			// Get SSO configuration
			startURL, _ := cmd.Flags().GetString("start-url")
			ssoRegion, _ := cmd.Flags().GetString("sso-region")

			// Try to find configuration if not provided
			if startURL == "" || ssoRegion == "" {
				instance, err := awsssolib.FindInstance("")
				if err != nil {
					return fmt.Errorf("no SSO configuration found. Please provide --start-url and --sso-region or set AWS_DEFAULT_SSO_START_URL and AWS_DEFAULT_SSO_REGION")
				}
				if startURL == "" {
					startURL = instance.StartURL
				}
				if ssoRegion == "" {
					ssoRegion = instance.Region
				}
			}

			// List available roles
			fmt.Fprintln(os.Stderr, "Fetching available accounts and roles...")
			roles, err := awsssolib.ListAvailableRoles(ctx, awsssolib.ListRolesInput{
				StartURL:  startURL,
				SSORegion: ssoRegion,
				Login:     true,
			})
			if err != nil {
				return fmt.Errorf("failed to list roles: %w", err)
			}

			if len(roles) == 0 {
				return fmt.Errorf("no roles available")
			}

			// Display available roles
			fmt.Fprintln(os.Stderr, "\nAvailable roles:")
			for i, role := range roles {
				fmt.Fprintf(os.Stderr, "[%d] %s - %s (%s)\n", i+1, role.AccountID, role.AccountName, role.RoleName)
			}

			// Prompt for selection
			reader := bufio.NewReader(os.Stdin)
			fmt.Fprint(os.Stderr, "\nSelect a role (enter number): ")
			input, err := reader.ReadString('\n')
			if err != nil {
				return err
			}

			var selection int
			_, err = fmt.Sscanf(strings.TrimSpace(input), "%d", &selection)
			if err != nil || selection < 1 || selection > len(roles) {
				return fmt.Errorf("invalid selection")
			}

			selectedRole := roles[selection-1]

			// If region not specified, prompt for it
			if region == "" {
				fmt.Fprint(os.Stderr, "AWS region (e.g., us-east-1): ")
				input, err = reader.ReadString('\n')
				if err != nil {
					return err
				}
				region = strings.TrimSpace(input)
			}

			// Load existing config
			config, err := awsssolib.LoadConfigFile("")
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Create profile
			profile := &awsssolib.Profile{
				Name:         profileName,
				StartURL:     startURL,
				SSORegion:    ssoRegion,
				AccountID:    selectedRole.AccountID,
				RoleName:     selectedRole.RoleName,
				Region:       region,
				OutputFormat: outputFormat,
			}

			// Add credential process if requested
			if credentialProcess {
				profile.CredProcess = fmt.Sprintf("aws-sso-util credential-process --profile %s", profileName)
			}

			// Save profile
			config.SetProfile(profile)
			err = config.SaveConfigFile("")
			if err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Fprintf(os.Stderr, "\nProfile '%s' configured successfully!\n", profileName)
			fmt.Fprintf(os.Stderr, "You can now use: aws --profile %s <command>\n", profileName)

			return nil
		},
	}

	cmd.Flags().StringVar(&region, "region", "", "AWS region for the profile")
	cmd.Flags().StringVar(&outputFormat, "output", "json", "Output format (json, text, table)")
	cmd.Flags().BoolVar(&credentialProcess, "credential-process", false, "Add credential process configuration")

	return cmd
}

// newConfigurePopulateCommand creates the configure populate command
func newConfigurePopulateCommand() *cobra.Command {
	var regions []string
	var profileTemplate string
	var credentialProcess bool
	var force bool

	cmd := &cobra.Command{
		Use:   "populate",
		Short: "Populate AWS CLI profiles for all available roles",
		Long: `Populate AWS CLI profiles for all available SSO roles.

This command creates profiles for all combinations of accounts, roles, and regions
you have access to through SSO.

Examples:
  # Populate profiles for one region
  aws-sso-util configure populate --regions us-east-1

  # Populate profiles for multiple regions
  aws-sso-util configure populate --regions us-east-1,us-west-2,eu-west-1

  # Use custom profile naming template
  aws-sso-util configure populate --regions us-east-1 --profile-template "{account_name}-{role_name}-{region}"

  # Force overwrite existing profiles
  aws-sso-util configure populate --regions us-east-1 --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			if len(regions) == 0 {
				return fmt.Errorf("at least one region must be specified with --regions")
			}

			// Get SSO configuration
			startURL, _ := cmd.Flags().GetString("start-url")
			ssoRegion, _ := cmd.Flags().GetString("sso-region")

			// Try to find configuration if not provided
			if startURL == "" || ssoRegion == "" {
				instance, err := awsssolib.FindInstance("")
				if err != nil {
					return fmt.Errorf("no SSO configuration found. Please provide --start-url and --sso-region or set AWS_DEFAULT_SSO_START_URL and AWS_DEFAULT_SSO_REGION")
				}
				if startURL == "" {
					startURL = instance.StartURL
				}
				if ssoRegion == "" {
					ssoRegion = instance.Region
				}
			}

			// List available accounts
			fmt.Fprintln(os.Stderr, "Fetching available accounts...")
			accounts, err := awsssolib.ListAvailableAccounts(ctx, awsssolib.ListAccountsInput{
				StartURL:  startURL,
				SSORegion: ssoRegion,
				Login:     true,
			})
			if err != nil {
				return fmt.Errorf("failed to list accounts: %w", err)
			}

			// List available roles
			fmt.Fprintln(os.Stderr, "Fetching available roles...")
			roles, err := awsssolib.ListAvailableRoles(ctx, awsssolib.ListRolesInput{
				StartURL:  startURL,
				SSORegion: ssoRegion,
			})
			if err != nil {
				return fmt.Errorf("failed to list roles: %w", err)
			}

			// Load existing config
			config, err := awsssolib.LoadConfigFile("")
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Create account map
			accountMap := make(map[string]*awsssolib.Account)
			for i := range accounts {
				accountMap[accounts[i].AccountID] = &accounts[i]
			}

			// Generate profiles
			profilesCreated := 0
			profilesSkipped := 0

			for _, role := range roles {
				account, ok := accountMap[role.AccountID]
				if !ok {
					continue
				}

				for _, region := range regions {
					// Generate profile name
					profileName := awsssolib.GenerateProfileName(profileTemplate, account, &role, region)

					// Check if profile exists
					if existing := config.GetProfile(profileName); existing != nil && !force {
						profilesSkipped++
						continue
					}

					// Create profile
					profile := &awsssolib.Profile{
						Name:      profileName,
						StartURL:  startURL,
						SSORegion: ssoRegion,
						AccountID: role.AccountID,
						RoleName:  role.RoleName,
						Region:    region,
					}

					// Add credential process if requested
					if credentialProcess {
						profile.CredProcess = fmt.Sprintf("aws-sso-util credential-process --profile %s", profileName)
					}

					config.SetProfile(profile)
					profilesCreated++
				}
			}

			// Save config
			err = config.SaveConfigFile("")
			if err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Fprintf(os.Stderr, "\nCreated %d profiles, skipped %d existing profiles\n", profilesCreated, profilesSkipped)

			return nil
		},
	}

	cmd.Flags().StringSliceVar(&regions, "regions", []string{}, "AWS regions to create profiles for (comma-separated)")
	cmd.Flags().StringVar(&profileTemplate, "profile-template", "", "Template for profile names (default: {account_name}.{role_name}.{region})")
	cmd.Flags().BoolVar(&credentialProcess, "credential-process", true, "Add credential process configuration")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing profiles")

	return cmd
}
