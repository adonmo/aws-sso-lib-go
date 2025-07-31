package awsssolib

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Default configuration file paths
var (
	DefaultAWSConfigFile      = filepath.Join(os.Getenv("HOME"), ".aws", "config")
	DefaultAWSCredentialsFile = filepath.Join(os.Getenv("HOME"), ".aws", "credentials")
)

// Profile represents an AWS CLI profile
type Profile struct {
	Name         string
	StartURL     string
	Region       string
	SSORegion    string
	AccountID    string
	RoleName     string
	CredProcess  string
	OutputFormat string
}

// ConfigFile represents AWS configuration
type ConfigFile struct {
	profiles map[string]*Profile
}

// NewConfigFile creates a new config file
func NewConfigFile() *ConfigFile {
	return &ConfigFile{
		profiles: make(map[string]*Profile),
	}
}

// LoadConfigFile loads AWS config from file
func LoadConfigFile(filename string) (*ConfigFile, error) {
	if filename == "" {
		filename = DefaultAWSConfigFile
	}

	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return NewConfigFile(), nil
		}
		return nil, err
	}
	defer file.Close()

	config := NewConfigFile()
	scanner := bufio.NewScanner(file)

	var currentProfile *Profile
	profileRegex := regexp.MustCompile(`^\[profile\s+(.+)\]$`)
	defaultRegex := regexp.MustCompile(`^\[default\]$`)
	keyValueRegex := regexp.MustCompile(`^\s*(\w+)\s*=\s*(.+)$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Check for profile header
		if matches := profileRegex.FindStringSubmatch(line); matches != nil {
			profileName := matches[1]
			currentProfile = &Profile{Name: profileName}
			config.profiles[profileName] = currentProfile
			continue
		}

		// Check for default profile
		if defaultRegex.MatchString(line) {
			currentProfile = &Profile{Name: "default"}
			config.profiles["default"] = currentProfile
			continue
		}

		// Parse key-value pairs
		if currentProfile != nil && keyValueRegex.MatchString(line) {
			matches := keyValueRegex.FindStringSubmatch(line)
			key := matches[1]
			value := strings.TrimSpace(matches[2])

			switch key {
			case "sso_start_url":
				currentProfile.StartURL = value
			case "sso_region":
				currentProfile.SSORegion = value
			case "sso_account_id":
				currentProfile.AccountID = value
			case "sso_role_name":
				currentProfile.RoleName = value
			case "region":
				currentProfile.Region = value
			case "credential_process":
				currentProfile.CredProcess = value
			case "output":
				currentProfile.OutputFormat = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return config, nil
}

// SaveConfigFile saves the config to file
func (c *ConfigFile) SaveConfigFile(filename string) error {
	if filename == "" {
		filename = DefaultAWSConfigFile
	}

	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	// Create temp file
	tempFile, err := os.CreateTemp(dir, ".config.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tempFile.Name())

	writer := bufio.NewWriter(tempFile)

	// Write profiles
	for name, profile := range c.profiles {
		if name == "default" {
			_, err = writer.WriteString("[default]\n")
		} else {
			_, err = writer.WriteString(fmt.Sprintf("[profile %s]\n", name))
		}
		if err != nil {
			return err
		}

		// Write profile properties
		if profile.StartURL != "" {
			_, err = writer.WriteString(fmt.Sprintf("sso_start_url = %s\n", profile.StartURL))
			if err != nil {
				return err
			}
		}
		if profile.SSORegion != "" {
			_, err = writer.WriteString(fmt.Sprintf("sso_region = %s\n", profile.SSORegion))
			if err != nil {
				return err
			}
		}
		if profile.AccountID != "" {
			_, err = writer.WriteString(fmt.Sprintf("sso_account_id = %s\n", profile.AccountID))
			if err != nil {
				return err
			}
		}
		if profile.RoleName != "" {
			_, err = writer.WriteString(fmt.Sprintf("sso_role_name = %s\n", profile.RoleName))
			if err != nil {
				return err
			}
		}
		if profile.Region != "" {
			_, err = writer.WriteString(fmt.Sprintf("region = %s\n", profile.Region))
			if err != nil {
				return err
			}
		}
		if profile.CredProcess != "" {
			_, err = writer.WriteString(fmt.Sprintf("credential_process = %s\n", profile.CredProcess))
			if err != nil {
				return err
			}
		}
		if profile.OutputFormat != "" {
			_, err = writer.WriteString(fmt.Sprintf("output = %s\n", profile.OutputFormat))
			if err != nil {
				return err
			}
		}

		_, err = writer.WriteString("\n")
		if err != nil {
			return err
		}
	}

	if err := writer.Flush(); err != nil {
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}

	// Rename temp file to actual file
	return os.Rename(tempFile.Name(), filename)
}

// GetProfile returns a profile by name
func (c *ConfigFile) GetProfile(name string) *Profile {
	return c.profiles[name]
}

// SetProfile adds or updates a profile
func (c *ConfigFile) SetProfile(profile *Profile) {
	c.profiles[profile.Name] = profile
}

// RemoveProfile removes a profile
func (c *ConfigFile) RemoveProfile(name string) {
	delete(c.profiles, name)
}

// ListProfiles returns all profile names
func (c *ConfigFile) ListProfiles() []string {
	names := make([]string, 0, len(c.profiles))
	for name := range c.profiles {
		names = append(names, name)
	}
	return names
}

// GetSSOProfiles returns all profiles with SSO configuration
func (c *ConfigFile) GetSSOProfiles() []*Profile {
	profiles := make([]*Profile, 0)
	for _, profile := range c.profiles {
		if profile.StartURL != "" && profile.SSORegion != "" {
			profiles = append(profiles, profile)
		}
	}
	return profiles
}

// FindInstance finds SSO instance configuration from environment or config
func FindInstance(profileName string) (*SSOInstance, error) {
	// Check environment variables first
	startURL := os.Getenv("AWS_DEFAULT_SSO_START_URL")
	region := os.Getenv("AWS_DEFAULT_SSO_REGION")

	if startURL != "" && region != "" {
		return &SSOInstance{
			StartURL:       startURL,
			Region:         region,
			StartURLSource: "environment",
			RegionSource:   "environment",
		}, nil
	}

	// Check profile if specified
	if profileName != "" {
		config, err := LoadConfigFile("")
		if err != nil {
			return nil, err
		}

		profile := config.GetProfile(profileName)
		if profile != nil && profile.StartURL != "" && profile.SSORegion != "" {
			return &SSOInstance{
				StartURL:       profile.StartURL,
				Region:         profile.SSORegion,
				StartURLSource: "profile",
				RegionSource:   "profile",
			}, nil
		}
	}

	// Check all profiles in config
	config, err := LoadConfigFile("")
	if err != nil {
		return nil, err
	}

	ssoProfiles := config.GetSSOProfiles()
	if len(ssoProfiles) > 0 {
		// Return the first SSO profile found
		profile := ssoProfiles[0]
		return &SSOInstance{
			StartURL:       profile.StartURL,
			Region:         profile.SSORegion,
			StartURLSource: "config",
			RegionSource:   "config",
		}, nil
	}

	return nil, fmt.Errorf("no SSO configuration found")
}

// GenerateProfileName generates a profile name based on a template
func GenerateProfileName(template string, account *Account, role *Role, region string) string {
	// Default template if empty
	if template == "" {
		template = "{account_name}.{role_name}.{region}"
	}

	// Replace placeholders
	name := template
	name = strings.ReplaceAll(name, "{account_id}", account.AccountID)
	name = strings.ReplaceAll(name, "{account_name}", sanitizeName(account.AccountName))
	name = strings.ReplaceAll(name, "{role_name}", sanitizeName(role.RoleName))
	name = strings.ReplaceAll(name, "{region}", region)

	// Clean up the name
	name = strings.ToLower(name)
	name = regexp.MustCompile(`[^a-z0-9._-]`).ReplaceAllString(name, "-")
	name = regexp.MustCompile(`-+`).ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")

	return name
}

// sanitizeName removes special characters from names
func sanitizeName(name string) string {
	// Remove common special characters
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, ":", "-")
	name = strings.ReplaceAll(name, "@", "-")

	// Remove multiple dashes
	name = regexp.MustCompile(`-+`).ReplaceAllString(name, "-")

	return name
}

// Validation constants and functions
var (
	// AWS Account ID regex (12 digits)
	accountIDRegex = regexp.MustCompile(`^\d{12}$`)
	// AWS region regex (pattern like us-east-1, eu-west-2, etc.)
	regionRegex = regexp.MustCompile(`^[a-z]{2}-[a-z]+-\d+$`)
	// Role name regex (alphanumeric, plus =,.@_- characters)
	roleNameRegex = regexp.MustCompile(`^[\w+=,.@_-]+$`)
)

// ValidateStartURL validates an SSO start URL
func ValidateStartURL(startURL string) error {
	if startURL == "" {
		return &InvalidConfigError{Message: "start URL cannot be empty"}
	}

	parsedURL, err := url.Parse(startURL)
	if err != nil {
		return &InvalidConfigError{Message: fmt.Sprintf("invalid start URL format: %v", err)}
	}

	if parsedURL.Scheme != "https" {
		return &InvalidConfigError{Message: "start URL must use HTTPS"}
	}

	if parsedURL.Host == "" {
		return &InvalidConfigError{Message: "start URL must have a valid host"}
	}

	// Check for common SSO URL patterns
	if !strings.Contains(parsedURL.Host, "awsapps.com") && !strings.Contains(parsedURL.Host, "signin.aws") {
		return &InvalidConfigError{Message: "start URL does not appear to be a valid AWS SSO URL"}
	}

	return nil
}

// ValidateRegion validates an AWS region
func ValidateRegion(region string) error {
	if region == "" {
		return &InvalidConfigError{Message: "region cannot be empty"}
	}

	if !regionRegex.MatchString(region) {
		return &InvalidConfigError{Message: fmt.Sprintf("invalid region format: %s", region)}
	}

	return nil
}

// ValidateAccountID validates an AWS account ID
func ValidateAccountID(accountID string) error {
	if accountID == "" {
		return &InvalidConfigError{Message: "account ID cannot be empty"}
	}

	// Remove any formatting (dashes, spaces)
	cleanID := formatAccountID(accountID)

	if !accountIDRegex.MatchString(cleanID) {
		return &InvalidConfigError{Message: fmt.Sprintf("invalid account ID format: %s (must be 12 digits)", accountID)}
	}

	return nil
}

// ValidateRoleName validates an AWS IAM role name
func ValidateRoleName(roleName string) error {
	if roleName == "" {
		return &InvalidConfigError{Message: "role name cannot be empty"}
	}

	if len(roleName) > 64 {
		return &InvalidConfigError{Message: fmt.Sprintf("role name too long: %d characters (max 64)", len(roleName))}
	}

	if !roleNameRegex.MatchString(roleName) {
		return &InvalidConfigError{Message: fmt.Sprintf("invalid role name format: %s", roleName)}
	}

	return nil
}

// ValidateProfile validates a complete profile configuration
func ValidateProfile(profile *Profile) error {
	if profile == nil {
		return &InvalidConfigError{Message: "profile cannot be nil"}
	}

	// Validate SSO-specific fields if present
	if profile.StartURL != "" || profile.SSORegion != "" || profile.AccountID != "" || profile.RoleName != "" {
		if err := ValidateStartURL(profile.StartURL); err != nil {
			return err
		}
		if err := ValidateRegion(profile.SSORegion); err != nil {
			return err
		}
		if err := ValidateAccountID(profile.AccountID); err != nil {
			return err
		}
		if err := ValidateRoleName(profile.RoleName); err != nil {
			return err
		}
	}

	// Validate region if present
	if profile.Region != "" {
		if err := ValidateRegion(profile.Region); err != nil {
			return err
		}
	}

	return nil
}

// ValidateGetAWSConfigInput validates input for GetAWSConfig
func ValidateGetAWSConfigInput(input GetAWSConfigInput) error {
	if err := ValidateStartURL(input.StartURL); err != nil {
		return err
	}
	if err := ValidateRegion(input.SSORegion); err != nil {
		return err
	}
	if err := ValidateAccountID(input.AccountID); err != nil {
		return err
	}
	if err := ValidateRoleName(input.RoleName); err != nil {
		return err
	}
	if err := ValidateRegion(input.Region); err != nil {
		return err
	}
	return nil
}

// ValidateLoginInput validates input for Login
func ValidateLoginInput(input LoginInput) error {
	if err := ValidateStartURL(input.StartURL); err != nil {
		return err
	}
	if err := ValidateRegion(input.SSORegion); err != nil {
		return err
	}
	return nil
}
