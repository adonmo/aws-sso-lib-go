package awsssolib

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// BrowserLauncher handles opening URLs in the user's browser
type BrowserLauncher struct {
	DisableBrowser bool
	CustomCommand  string
}

// NewBrowserLauncher creates a new browser launcher
func NewBrowserLauncher(disableBrowser bool) *BrowserLauncher {
	return &BrowserLauncher{
		DisableBrowser: disableBrowser,
	}
}

// OpenURL attempts to open a URL in the user's default browser
func (b *BrowserLauncher) OpenURL(url string) error {
	if b.DisableBrowser {
		return nil
	}

	// Check if AWS_SSO_DISABLE_BROWSER is set
	if os.Getenv("AWS_SSO_DISABLE_BROWSER") == "1" || os.Getenv("AWS_SSO_DISABLE_BROWSER") == "true" {
		return nil
	}

	if b.CustomCommand != "" {
		return b.openWithCustomCommand(url)
	}

	return b.openWithDefaultBrowser(url)
}

// openWithDefaultBrowser opens URL using the OS default browser
func (b *BrowserLauncher) openWithDefaultBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "linux":
		// Try different commands in order of preference
		commands := [][]string{
			{"xdg-open", url},
			{"sensible-browser", url},
			{"x-www-browser", url},
			{"firefox", url},
			{"chromium", url},
			{"google-chrome", url},
		}
		
		for _, args := range commands {
			if _, err := exec.LookPath(args[0]); err == nil {
				cmd = exec.Command(args[0], args[1:]...)
				break
			}
		}
		
		if cmd == nil {
			return fmt.Errorf("no suitable browser found")
		}
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	return cmd.Start()
}

// openWithCustomCommand opens URL using a custom command
func (b *BrowserLauncher) openWithCustomCommand(url string) error {
	// Replace {url} placeholder with actual URL
	command := strings.ReplaceAll(b.CustomCommand, "{url}", url)
	
	// Split command into executable and arguments
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("empty custom browser command")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	return cmd.Start()
}

// DefaultAuthHandler provides the default interactive authentication handler
func DefaultAuthHandler(ctx context.Context, params AuthHandlerParams) error {
	launcher := NewBrowserLauncher(false)
	
	// Try to open browser
	browserErr := launcher.OpenURL(params.VerificationURIComplete)
	
	// Always print the manual instructions
	fmt.Fprintf(os.Stderr, "\n")
	if browserErr != nil {
		fmt.Fprintf(os.Stderr, "Failed to open browser automatically.\n")
	}
	
	fmt.Fprintf(os.Stderr, "Attempting to open the SSO authorization page in your default browser.\n")
	fmt.Fprintf(os.Stderr, "If the browser does not open or you wish to use a different device to authorize this request, open the following URL:\n\n")
	fmt.Fprintf(os.Stderr, "\t%s\n\n", params.VerificationURI)
	fmt.Fprintf(os.Stderr, "Then enter the code:\n\n")
	fmt.Fprintf(os.Stderr, "\t%s\n\n", params.UserCode)
	
	// Calculate time remaining
	remaining := time.Until(params.ExpiresAt)
	fmt.Fprintf(os.Stderr, "The code will expire in %d minutes.\n", int(remaining.Minutes()))
	
	return nil
}

// NonInteractiveAuthHandler returns an error indicating authentication is needed
func NonInteractiveAuthHandler(ctx context.Context, params AuthHandlerParams) error {
	return &AuthenticationNeededError{
		Message: fmt.Sprintf("authentication required - visit %s and enter code %s", 
			params.VerificationURI, params.UserCode),
	}
}