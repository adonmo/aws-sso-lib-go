package awsssolib

import (
	"os"
	"testing"
	"time"
)

func TestMemoryCache(t *testing.T) {
	cache := NewMemoryCache()

	// Test Put and Get
	key := "test-key"
	data := []byte("test-data")

	err := cache.Put(key, data)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	retrieved, err := cache.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(retrieved) != string(data) {
		t.Errorf("Expected %s, got %s", string(data), string(retrieved))
	}

	// Test Delete
	err = cache.Delete(key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	retrieved, err = cache.Get(key)
	if err != nil {
		t.Fatalf("Get after delete failed: %v", err)
	}
	if retrieved != nil {
		t.Error("Expected nil after delete")
	}
}

func TestFileCache(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "cache-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache := NewFileCache(tempDir)

	// Test Put and Get
	key := "test-key"
	data := []byte("test-data")

	err = cache.Put(key, data)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	retrieved, err := cache.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(retrieved) != string(data) {
		t.Errorf("Expected %s, got %s", string(data), string(retrieved))
	}

	// Test Delete
	err = cache.Delete(key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	retrieved, err = cache.Get(key)
	if err != nil {
		t.Fatalf("Get after delete failed: %v", err)
	}
	if retrieved != nil {
		t.Error("Expected nil after delete")
	}
}

func TestTokenCaching(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "token-cache-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache := NewFileCache(tempDir)
	startURL := "https://test.awsapps.com/start"

	// Test token caching
	token := &Token{
		AccessToken: "test-access-token",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
		StartURL:    startURL,
		Region:      "us-east-1",
	}

	err = PutCachedToken(cache, startURL, token)
	if err != nil {
		t.Fatalf("PutCachedToken failed: %v", err)
	}

	retrieved, err := GetCachedToken(cache, startURL)
	if err != nil {
		t.Fatalf("GetCachedToken failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected token, got nil")
	}

	if retrieved.AccessToken != token.AccessToken {
		t.Errorf("Expected access token %s, got %s", token.AccessToken, retrieved.AccessToken)
	}

	// Test expired token
	expiredToken := &Token{
		AccessToken: "expired-token",
		ExpiresAt:   time.Now().Add(-1 * time.Hour), // Expired
		StartURL:    startURL,
		Region:      "us-east-1",
	}

	err = PutCachedToken(cache, startURL, expiredToken)
	if err != nil {
		t.Fatalf("PutCachedToken failed: %v", err)
	}

	retrieved, err = GetCachedToken(cache, startURL)
	if err != nil {
		t.Fatalf("GetCachedToken failed: %v", err)
	}

	if retrieved != nil {
		t.Error("Expected nil for expired token")
	}
}

func TestGenerateProfileName(t *testing.T) {
	account := &Account{
		AccountID:   "123456789012",
		AccountName: "Test Account",
	}
	role := &Role{
		RoleName: "Test_Role",
	}
	region := "us-west-2"

	tests := []struct {
		template string
		expected string
	}{
		{
			template: "",
			expected: "test-account.test-role.us-west-2",
		},
		{
			template: "{account_id}-{role_name}",
			expected: "123456789012-test-role",
		},
		{
			template: "{account_name}_{region}",
			expected: "test-account_us-west-2",
		},
	}

	for _, tt := range tests {
		result := GenerateProfileName(tt.template, account, role, region)
		if result != tt.expected {
			t.Errorf("Template %s: expected %s, got %s", tt.template, tt.expected, result)
		}
	}
}

func TestFormatAccountID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"123456789012", "123456789012"},
		{"1234-5678-9012", "123456789012"},
		{"1234 5678 9012", "123456789012"},
		{"123-456-789-012", "123456789012"},
	}

	for _, tt := range tests {
		result := formatAccountID(tt.input)
		if result != tt.expected {
			t.Errorf("Input %s: expected %s, got %s", tt.input, tt.expected, result)
		}
	}
}