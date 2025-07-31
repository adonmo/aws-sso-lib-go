package awsssolib

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	logger := getLogger(input.Config)

	logger.Debug("Starting AWS config retrieval",
		slog.String("start_url", input.StartURL),
		slog.String("sso_region", input.SSORegion),
		slog.String("account_id", input.AccountID),
		slog.String("role_name", input.RoleName),
		slog.String("region", input.Region),
		slog.Bool("login", input.Login))

	// Validate input using centralized validation
	if err := ValidateGetAWSConfigInput(input); err != nil {
		logger.Error("AWS config input validation failed", slog.Any("error", err))
		return aws.Config{}, err
	}

	// Format account ID (remove dashes if present)
	accountID := formatAccountID(input.AccountID)

	// Login if requested
	if input.Login {
		logger.Info("Performing SSO login before config retrieval")
		_, err := Login(ctx, LoginInput{
			StartURL:  input.StartURL,
			SSORegion: input.SSORegion,
			SSOCache:  input.SSOCache,
			Config:    input.Config,
		})
		if err != nil {
			logger.Error("SSO login failed", slog.Any("error", err))
			return aws.Config{}, fmt.Errorf("login failed: %w", err)
		}
		logger.Info("SSO login completed successfully")
	}

	// Create credential provider
	provider := &ssoCredentialProvider{
		startURL:        input.StartURL,
		ssoRegion:       input.SSORegion,
		accountID:       accountID,
		roleName:        input.RoleName,
		ssoCache:        input.SSOCache,
		credentialCache: input.CredentialCache,
		config:          input.Config,
	}

	// Create AWS config
	logger.Debug("Creating AWS SDK configuration")
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(input.Region),
		config.WithCredentialsProvider(provider),
	)
	if err != nil {
		logger.Error("Failed to load AWS configuration", slog.Any("error", err))
		return aws.Config{}, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	logger.Info("AWS configuration created successfully",
		slog.String("region", input.Region),
		slog.String("account_id", accountID),
		slog.String("role_name", input.RoleName))
	return cfg, nil
}

// Login performs SSO login and returns the access token
func Login(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	logger := getLogger(input.Config)

	logger.Info("Starting SSO login",
		slog.String("start_url", input.StartURL),
		slog.String("sso_region", input.SSORegion),
		slog.Bool("force_refresh", input.ForceRefresh),
		slog.Bool("disable_browser", input.DisableBrowser))

	// Validate input using centralized validation
	if err := ValidateLoginInput(input); err != nil {
		logger.Error("Login input validation failed", slog.Any("error", err))
		return nil, err
	}

	// Check for existing token if not forcing refresh
	if !input.ForceRefresh {
		logger.Debug("Checking for cached SSO token")
		token, err := GetCachedToken(input.SSOCache, input.StartURL)
		if err == nil && token != nil {
			// Check if token is still valid with expiry window
			expiryWindow := input.ExpiryWindow
			if expiryWindow == 0 {
				expiryWindow = defaultExpiryWindow
			}

			if time.Now().Add(expiryWindow).Before(token.ExpiresAt) {
				logger.Info("Using cached SSO token",
					slog.Time("expires_at", token.ExpiresAt),
					slog.Duration("expires_in", time.Until(token.ExpiresAt)))
				return &LoginOutput{
					Token:     token,
					ExpiresAt: token.ExpiresAt,
				}, nil
			} else {
				logger.Debug("Cached token is expired or will expire soon",
					slog.Time("expires_at", token.ExpiresAt),
					slog.Duration("expiry_window", expiryWindow))
			}
		} else if err != nil {
			logger.Debug("Failed to retrieve cached token", slog.Any("error", err))
		} else {
			logger.Debug("No cached token found")
		}
	}

	// Perform device authorization flow
	logger.Info("Starting device authorization flow")
	token, err := performDeviceAuthorization(ctx, input)
	if err != nil {
		logger.Error("Device authorization failed", slog.Any("error", err))
		return nil, err
	}
	logger.Info("Device authorization completed successfully",
		slog.Time("expires_at", token.ExpiresAt))

	// Cache the token
	logger.Debug("Caching SSO token")
	if err := PutCachedToken(input.SSOCache, input.StartURL, token); err != nil {
		// Log error but don't fail - token caching is not critical
		logger.Warn("Failed to cache SSO token", slog.Any("error", err))
	} else {
		logger.Debug("SSO token cached successfully")
	}

	logger.Info("SSO login completed successfully")
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
		// Continue with cache deletion even if API call fails
		// Note: In production, this should use structured logging
		_ = err // Silently ignore API errors to avoid exposing internal details
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
				// Note: In production, this should use structured logging
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
		return nil, fmt.Errorf("failed to register SSO client: %w", err)
	}

	// Start device authorization
	authResp, err := oidcClient.StartDeviceAuthorization(ctx, &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     registerResp.ClientId,
		ClientSecret: registerResp.ClientSecret,
		StartUrl:     aws.String(input.StartURL),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start SSO device authorization: %w", err)
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

	// Poll for token with timeout
	interval := time.Duration(authResp.Interval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Set a reasonable timeout for the entire authorization process (10 minutes)
	timeout := 10 * time.Minute
	if deadline, ok := ctx.Deadline(); ok {
		// Use context deadline if it's sooner than our timeout
		if time.Until(deadline) < timeout {
			timeout = time.Until(deadline)
		}
	}

	authCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-authCtx.Done():
			if authCtx.Err() == context.DeadlineExceeded {
				return nil, fmt.Errorf("SSO authorization timed out after %v", timeout)
			}
			return nil, authCtx.Err()
		case <-ticker.C:
			tokenResp, err := oidcClient.CreateToken(authCtx, &ssooidc.CreateTokenInput{
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
					// Authorization is still pending, continue polling silently
					continue
				} else if errors.As(err, &slowDownErr) {
					// Slow down the polling as requested by the server
					time.Sleep(time.Duration(authResp.Interval) * time.Second)
					continue
				} else if strings.Contains(err.Error(), "AuthorizationPendingException") {
					// Fallback string check for older SDK versions
					continue
				}
				return nil, fmt.Errorf("failed to obtain access token: %w", err)
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
	config          *Config
}

// Retrieve fetches credentials
func (p *ssoCredentialProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	logger := getLogger(p.config)

	logger.Debug("Starting credential retrieval",
		slog.String("account_id", p.accountID),
		slog.String("role_name", p.roleName),
		slog.String("sso_region", p.ssoRegion))

	// Add timeout for credential retrieval if not already set
	var retrieveCtx context.Context
	var cancel context.CancelFunc

	if _, ok := ctx.Deadline(); !ok {
		// No deadline set, add a reasonable timeout (30 seconds)
		retrieveCtx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	} else {
		retrieveCtx = ctx
	}
	// Check credential cache first
	cacheKey := generateCredentialCacheKey(p.startURL, p.accountID, p.roleName)
	if p.credentialCache != nil {
		logger.Debug("Checking credential cache")
		cached, err := GetCachedCredentials(p.credentialCache, cacheKey)
		if err == nil && cached != nil {
			logger.Info("Using cached credentials",
				slog.Time("expires_at", cached.Expiration),
				slog.Duration("expires_in", time.Until(cached.Expiration)))
			return aws.Credentials{
				AccessKeyID:     cached.AccessKeyID,
				SecretAccessKey: cached.SecretAccessKey,
				SessionToken:    cached.SessionToken,
				CanExpire:       true,
				Expires:         cached.Expiration,
				Source:          "SSO",
			}, nil
		} else if err != nil {
			logger.Debug("Failed to retrieve cached credentials", slog.Any("error", err))
		} else {
			logger.Debug("No cached credentials found")
		}
	}

	// Get SSO token
	logger.Debug("Retrieving SSO token")
	token, err := GetCachedToken(p.ssoCache, p.startURL)
	if err != nil || token == nil {
		logger.Error("SSO token not available", slog.Any("error", err))
		return aws.Credentials{}, &AuthenticationNeededError{}
	}
	logger.Debug("SSO token retrieved successfully")

	// Create SSO client
	logger.Debug("Creating SSO client")
	cfg, err := config.LoadDefaultConfig(retrieveCtx, config.WithRegion(p.ssoRegion))
	if err != nil {
		logger.Error("Failed to load AWS config for SSO client", slog.Any("error", err))
		return aws.Credentials{}, fmt.Errorf("failed to load config: %w", err)
	}

	client := sso.NewFromConfig(cfg)

	// Get role credentials
	logger.Debug("Calling SSO GetRoleCredentials API")
	resp, err := client.GetRoleCredentials(retrieveCtx, &sso.GetRoleCredentialsInput{
		AccessToken: aws.String(token.AccessToken),
		AccountId:   aws.String(p.accountID),
		RoleName:    aws.String(p.roleName),
	})
	if err != nil {
		logger.Error("Failed to get role credentials from SSO", slog.Any("error", err))
		return aws.Credentials{}, fmt.Errorf("failed to get role credentials: %w", err)
	}

	creds := resp.RoleCredentials
	expiration := time.Unix(creds.Expiration/1000, 0)

	logger.Info("Role credentials retrieved successfully",
		slog.Time("expires_at", expiration),
		slog.Duration("expires_in", time.Until(expiration)))

	// Cache credentials
	if p.credentialCache != nil {
		logger.Debug("Caching role credentials")
		cachedCreds := &CachedCredentials{
			AccessKeyID:     aws.ToString(creds.AccessKeyId),
			SecretAccessKey: aws.ToString(creds.SecretAccessKey),
			SessionToken:    aws.ToString(creds.SessionToken),
			Expiration:      expiration,
		}
		if err := PutCachedCredentials(p.credentialCache, cacheKey, cachedCreds); err != nil {
			logger.Warn("Failed to cache credentials", slog.Any("error", err))
		} else {
			logger.Debug("Credentials cached successfully")
		}
	}

	logger.Debug("Credential retrieval completed successfully")
	return aws.Credentials{
		AccessKeyID:     aws.ToString(creds.AccessKeyId),
		SecretAccessKey: aws.ToString(creds.SecretAccessKey),
		SessionToken:    aws.ToString(creds.SessionToken),
		CanExpire:       true,
		Expires:         expiration,
		Source:          "SSO",
	}, nil
}
