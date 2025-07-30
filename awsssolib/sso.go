package awsssolib

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc/types"
)

const (
	// Default SSO client registration
	defaultClientName = "aws-sso-lib-go"
	defaultClientType = "public"
	
	// Token expiry window (5 minutes)
	defaultExpiryWindow = 5 * time.Minute
)

// GetAWSConfig returns an AWS SDK v2 config for the specified account and role
func GetAWSConfig(ctx context.Context, input GetAWSConfigInput) (aws.Config, error) {
	// Validate input
	if input.StartURL == "" {
		return aws.Config{}, &InvalidConfigError{Message: "start URL is required"}
	}
	if input.SSORegion == "" {
		return aws.Config{}, &InvalidConfigError{Message: "SSO region is required"}
	}
	if input.AccountID == "" {
		return aws.Config{}, &InvalidConfigError{Message: "account ID is required"}
	}
	if input.RoleName == "" {
		return aws.Config{}, &InvalidConfigError{Message: "role name is required"}
	}
	if input.Region == "" {
		return aws.Config{}, &InvalidConfigError{Message: "region is required"}
	}

	// Format account ID (remove dashes if present)
	accountID := formatAccountID(input.AccountID)

	// Login if requested
	if input.Login {
		_, err := Login(ctx, LoginInput{
			StartURL:  input.StartURL,
			SSORegion: input.SSORegion,
			SSOCache:  input.SSOCache,
		})
		if err != nil {
			return aws.Config{}, fmt.Errorf("login failed: %w", err)
		}
	}

	// Create credential provider
	provider := &ssoCredentialProvider{
		startURL:        input.StartURL,
		ssoRegion:       input.SSORegion,
		accountID:       accountID,
		roleName:        input.RoleName,
		ssoCache:        input.SSOCache,
		credentialCache: input.CredentialCache,
	}

	// Create AWS config
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(input.Region),
		config.WithCredentialsProvider(provider),
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load config: %w", err)
	}

	return cfg, nil
}

// Login performs SSO login and returns the access token
func Login(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	// Validate input
	if input.StartURL == "" {
		return nil, &InvalidConfigError{Message: "start URL is required"}
	}
	if input.SSORegion == "" {
		return nil, &InvalidConfigError{Message: "SSO region is required"}
	}

	// Check for existing token if not forcing refresh
	if !input.ForceRefresh {
		token, err := GetCachedToken(input.SSOCache, input.StartURL)
		if err == nil && token != nil {
			// Check if token is still valid with expiry window
			expiryWindow := input.ExpiryWindow
			if expiryWindow == 0 {
				expiryWindow = defaultExpiryWindow
			}
			
			if time.Now().Add(expiryWindow).Before(token.ExpiresAt) {
				return &LoginOutput{
					Token:     token,
					ExpiresAt: token.ExpiresAt,
				}, nil
			}
		}
	}

	// Perform device authorization flow
	token, err := performDeviceAuthorization(ctx, input)
	if err != nil {
		return nil, err
	}

	// Cache the token
	if err := PutCachedToken(input.SSOCache, input.StartURL, token); err != nil {
		// Log error but don't fail
		fmt.Printf("Warning: failed to cache token: %v\n", err)
	}

	return &LoginOutput{
		Token:     token,
		ExpiresAt: token.ExpiresAt,
	}, nil
}

// Logout removes the cached SSO token
func Logout(ctx context.Context, startURL, ssoRegion string, ssoCache Cache) error {
	// Get the cached token
	token, err := GetCachedToken(ssoCache, startURL)
	if err != nil || token == nil {
		return nil // Already logged out
	}

	// Create SSO client
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(ssoRegion))
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	client := sso.NewFromConfig(cfg)

	// Call logout API
	_, err = client.Logout(ctx, &sso.LogoutInput{
		AccessToken: aws.String(token.AccessToken),
	})
	if err != nil {
		// Log error but continue with cache deletion
		fmt.Printf("Warning: SSO logout API call failed: %v\n", err)
	}

	// Delete cached token
	return DeleteCachedToken(ssoCache, startURL)
}

// ListAvailableAccounts returns all accounts accessible through SSO
func ListAvailableAccounts(ctx context.Context, input ListAccountsInput) ([]Account, error) {
	// Get token
	token, err := getTokenForOperation(ctx, input.StartURL, input.SSORegion, input.Login, input.SSOCache)
	if err != nil {
		return nil, err
	}

	// Create SSO client
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(input.SSORegion))
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	client := sso.NewFromConfig(cfg)

	// List accounts
	var accounts []Account
	var nextToken *string

	for {
		resp, err := client.ListAccounts(ctx, &sso.ListAccountsInput{
			AccessToken: aws.String(token.AccessToken),
			NextToken:   nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list accounts: %w", err)
		}

		for _, acc := range resp.AccountList {
			accounts = append(accounts, Account{
				AccountID:    aws.ToString(acc.AccountId),
				AccountName:  aws.ToString(acc.AccountName),
				EmailAddress: aws.ToString(acc.EmailAddress),
			})
		}

		nextToken = resp.NextToken
		if nextToken == nil {
			break
		}
	}

	return accounts, nil
}

// ListAvailableRoles returns all roles accessible through SSO
func ListAvailableRoles(ctx context.Context, input ListRolesInput) ([]Role, error) {
	// Get token
	token, err := getTokenForOperation(ctx, input.StartURL, input.SSORegion, input.Login, input.SSOCache)
	if err != nil {
		return nil, err
	}

	// Create SSO client
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(input.SSORegion))
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	client := sso.NewFromConfig(cfg)

	// Get accounts to iterate over
	var accountsToCheck []Account
	
	if len(input.AccountIDs) > 0 {
		// Use specified accounts
		for _, id := range input.AccountIDs {
			accountsToCheck = append(accountsToCheck, Account{
				AccountID:   formatAccountID(id),
				AccountName: "UNKNOWN",
			})
		}
	} else {
		// List all accounts
		accounts, err := ListAvailableAccounts(ctx, ListAccountsInput{
			StartURL:  input.StartURL,
			SSORegion: input.SSORegion,
			SSOCache:  input.SSOCache,
		})
		if err != nil {
			return nil, err
		}
		accountsToCheck = accounts
	}

	// List roles for each account
	var roles []Role
	
	for _, account := range accountsToCheck {
		var nextToken *string
		
		for {
			resp, err := client.ListAccountRoles(ctx, &sso.ListAccountRolesInput{
				AccessToken: aws.String(token.AccessToken),
				AccountId:   aws.String(account.AccountID),
				NextToken:   nextToken,
			})
			if err != nil {
				// Skip this account if we can't list roles
				fmt.Printf("Warning: failed to list roles for account %s: %v\n", account.AccountID, err)
				break
			}

			for _, role := range resp.RoleList {
				roles = append(roles, Role{
					RoleName:    aws.ToString(role.RoleName),
					AccountID:   account.AccountID,
					AccountName: account.AccountName,
				})
			}

			nextToken = resp.NextToken
			if nextToken == nil {
				break
			}
		}
	}

	return roles, nil
}

// performDeviceAuthorization performs the SSO device authorization flow
func performDeviceAuthorization(ctx context.Context, input LoginInput) (*Token, error) {
	// Create OIDC client
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(input.SSORegion))
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	oidcClient := ssooidc.NewFromConfig(cfg)

	// Register client
	registerResp, err := oidcClient.RegisterClient(ctx, &ssooidc.RegisterClientInput{
		ClientName: aws.String(defaultClientName),
		ClientType: aws.String(defaultClientType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register client: %w", err)
	}

	// Start device authorization
	authResp, err := oidcClient.StartDeviceAuthorization(ctx, &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     registerResp.ClientId,
		ClientSecret: registerResp.ClientSecret,
		StartUrl:     aws.String(input.StartURL),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start device authorization: %w", err)
	}

	// Call auth handler
	authHandler := input.UserAuthHandler
	if authHandler == nil {
		if input.DisableBrowser {
			authHandler = NonInteractiveAuthHandler
		} else {
			authHandler = DefaultAuthHandler
		}
	}

	expiresAt := time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second)
	err = authHandler(ctx, AuthHandlerParams{
		VerificationURI:         aws.ToString(authResp.VerificationUri),
		UserCode:                aws.ToString(authResp.UserCode),
		VerificationURIComplete: aws.ToString(authResp.VerificationUriComplete),
		ExpiresAt:               expiresAt,
	})
	if err != nil {
		return nil, err
	}

	// Poll for token
	interval := time.Duration(authResp.Interval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			tokenResp, err := oidcClient.CreateToken(ctx, &ssooidc.CreateTokenInput{
				ClientId:     registerResp.ClientId,
				ClientSecret: registerResp.ClientSecret,
				DeviceCode:   authResp.DeviceCode,
				GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
			})
			
			if err != nil {
				// Check if it's an authorization pending error
				var authPendingErr *types.AuthorizationPendingException
				var slowDownErr *types.SlowDownException
				
				if errors.As(err, &authPendingErr) {
					// Authorization is still pending, continue polling
					fmt.Printf("Waiting for authorization... (polling every %d seconds)\n", authResp.Interval)
					continue
				} else if errors.As(err, &slowDownErr) {
					// Slow down the polling as requested by the server
					fmt.Printf("Slowing down polling as requested by server...\n")
					time.Sleep(time.Duration(authResp.Interval) * time.Second)
					continue
				} else if strings.Contains(err.Error(), "AuthorizationPendingException") {
					// Fallback string check for older SDK versions
					fmt.Printf("Waiting for authorization... (polling every %d seconds)\n", authResp.Interval)
					continue
				}
				return nil, fmt.Errorf("failed to create token: %w", err)
			}

			// Success! Create token object
			token := &Token{
				AccessToken:      aws.ToString(tokenResp.AccessToken),
				ExpiresAt:        time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
				RefreshToken:     aws.ToString(tokenResp.RefreshToken),
				ClientID:         aws.ToString(registerResp.ClientId),
				ClientSecret:     aws.ToString(registerResp.ClientSecret),
				RegistrationTime: time.Now(),
				Region:           input.SSORegion,
				StartURL:         input.StartURL,
			}

			return token, nil
		}
	}
}

// getTokenForOperation gets a token for an operation, optionally logging in
func getTokenForOperation(ctx context.Context, startURL, ssoRegion string, login bool, ssoCache Cache) (*Token, error) {
	// Try to get cached token
	token, err := GetCachedToken(ssoCache, startURL)
	if err == nil && token != nil {
		return token, nil
	}

	// If login is enabled, try to log in
	if login {
		output, err := Login(ctx, LoginInput{
			StartURL:  startURL,
			SSORegion: ssoRegion,
			SSOCache:  ssoCache,
		})
		if err != nil {
			return nil, err
		}
		return output.Token, nil
	}

	// No token and login not enabled
	return nil, &AuthenticationNeededError{}
}

// formatAccountID formats an account ID by removing dashes
func formatAccountID(accountID string) string {
	result := ""
	for _, r := range accountID {
		if r >= '0' && r <= '9' {
			result += string(r)
		}
	}
	return result
}

// ssoCredentialProvider implements AWS SDK v2 CredentialsProvider
type ssoCredentialProvider struct {
	startURL        string
	ssoRegion       string
	accountID       string
	roleName        string
	ssoCache        Cache
	credentialCache Cache
}

// Retrieve fetches credentials
func (p *ssoCredentialProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	// Check credential cache first
	cacheKey := generateCredentialCacheKey(p.startURL, p.accountID, p.roleName)
	if p.credentialCache != nil {
		cached, err := GetCachedCredentials(p.credentialCache, cacheKey)
		if err == nil && cached != nil {
			return aws.Credentials{
				AccessKeyID:     cached.AccessKeyID,
				SecretAccessKey: cached.SecretAccessKey,
				SessionToken:    cached.SessionToken,
				CanExpire:       true,
				Expires:         cached.Expiration,
				Source:          "SSO",
			}, nil
		}
	}

	// Get SSO token
	token, err := GetCachedToken(p.ssoCache, p.startURL)
	if err != nil || token == nil {
		return aws.Credentials{}, &AuthenticationNeededError{}
	}

	// Create SSO client
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(p.ssoRegion))
	if err != nil {
		return aws.Credentials{}, fmt.Errorf("failed to load config: %w", err)
	}

	client := sso.NewFromConfig(cfg)

	// Get role credentials
	resp, err := client.GetRoleCredentials(ctx, &sso.GetRoleCredentialsInput{
		AccessToken: aws.String(token.AccessToken),
		AccountId:   aws.String(p.accountID),
		RoleName:    aws.String(p.roleName),
	})
	if err != nil {
		return aws.Credentials{}, fmt.Errorf("failed to get role credentials: %w", err)
	}

	creds := resp.RoleCredentials
	expiration := time.Unix(creds.Expiration/1000, 0)

	// Cache credentials
	if p.credentialCache != nil {
		cachedCreds := &CachedCredentials{
			AccessKeyID:     aws.ToString(creds.AccessKeyId),
			SecretAccessKey: aws.ToString(creds.SecretAccessKey),
			SessionToken:    aws.ToString(creds.SessionToken),
			Expiration:      expiration,
		}
		_ = PutCachedCredentials(p.credentialCache, cacheKey, cachedCreds)
	}

	return aws.Credentials{
		AccessKeyID:     aws.ToString(creds.AccessKeyId),
		SecretAccessKey: aws.ToString(creds.SecretAccessKey),
		SessionToken:    aws.ToString(creds.SessionToken),
		CanExpire:       true,
		Expires:         expiration,
		Source:          "SSO",
	}, nil
}