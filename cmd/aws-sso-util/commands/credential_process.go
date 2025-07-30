package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/adonmo/aws-sso-lib-go/awsssolib"
	"github.com/spf13/cobra"
)

// CredentialProcessOutput represents the output format for credential_process
type CredentialProcessOutput struct {
	Version         int    `json:"Version"`
	AccessKeyID     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
	SessionToken    string `json:"SessionToken,omitempty"`
	Expiration      string `json:"Expiration,omitempty"`
}

// NewCredentialProcessCommand creates the credential-process command
func NewCredentialProcessCommand() *cobra.Command {
	var profileName string
	var accountID string
	var roleName string
	var startURL string
	var ssoRegion string

	cmd := &cobra.Command{
		Use:    "credential-process",
		Short:  "Output credentials in credential_process format",
		Long:   `Output AWS credentials in the format expected by the credential_process configuration.`,
		Hidden: true, // Hide from main help as it's meant to be used by AWS CLI
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// If profile is specified, load configuration from it
			if profileName != "" {
				config, err := awsssolib.LoadConfigFile("")
				if err != nil {
					return err
				}

				profile := config.GetProfile(profileName)
				if profile == nil {
					return fmt.Errorf("profile '%s' not found", profileName)
				}

				// Override with profile values
				if profile.StartURL != "" {
					startURL = profile.StartURL
				}
				if profile.SSORegion != "" {
					ssoRegion = profile.SSORegion
				}
				if profile.AccountID != "" {
					accountID = profile.AccountID
				}
				if profile.RoleName != "" {
					roleName = profile.RoleName
				}
			}

			// Validate required parameters
			if startURL == "" || ssoRegion == "" || accountID == "" || roleName == "" {
				return fmt.Errorf("missing required SSO configuration")
			}

			// Get AWS config
			cfg, err := awsssolib.GetAWSConfig(ctx, awsssolib.GetAWSConfigInput{
				StartURL:  startURL,
				SSORegion: ssoRegion,
				AccountID: accountID,
				RoleName:  roleName,
				Region:    "us-east-1", // Region doesn't matter for credentials
				Login:     false,       // Don't try to login interactively
			})
			if err != nil {
				return err
			}

			// Get credentials
			creds, err := cfg.Credentials.Retrieve(ctx)
			if err != nil {
				return err
			}

			// Create output
			output := CredentialProcessOutput{
				Version:         1,
				AccessKeyID:     creds.AccessKeyID,
				SecretAccessKey: creds.SecretAccessKey,
				SessionToken:    creds.SessionToken,
			}

			// Add expiration if available
			if creds.CanExpire && !creds.Expires.IsZero() {
				output.Expiration = creds.Expires.Format("2006-01-02T15:04:05Z")
			}

			// Output JSON
			encoder := json.NewEncoder(os.Stdout)
			return encoder.Encode(output)
		},
	}

	cmd.Flags().StringVar(&profileName, "profile", "", "AWS profile name")
	cmd.Flags().StringVar(&accountID, "account", "", "AWS account ID")
	cmd.Flags().StringVar(&roleName, "role", "", "SSO role name")
	cmd.Flags().StringVar(&startURL, "start-url", "", "SSO start URL")
	cmd.Flags().StringVar(&ssoRegion, "sso-region", "", "SSO region")

	return cmd
}