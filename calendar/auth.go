package calendar

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"time"

	"meetingbar/config"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	oauth2api "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

const (
	// OAuth2 scopes required for calendar access
	CalendarScope = "https://www.googleapis.com/auth/calendar.readonly"
	UserInfoScope = "https://www.googleapis.com/auth/userinfo.email"
)

var (
	oauth2Config *oauth2.Config
)

func init() {
	oauth2Config = &oauth2.Config{
		RedirectURL:  "http://localhost:8080/callback",
		Scopes: []string{
			CalendarScope,
			UserInfoScope,
		},
		Endpoint: google.Endpoint,
	}
}

func SetOAuth2Config(clientID, clientSecret string) {
	oauth2Config.ClientID = clientID
	oauth2Config.ClientSecret = clientSecret
}

func StartOAuth2Flow(ctx context.Context, cfg *config.Config) (*config.Account, error) {
	// Update OAuth2 config with stored credentials
	if cfg.OAuth2.ClientID == "" || cfg.OAuth2.ClientSecret == "" {
		return nil, fmt.Errorf("OAuth2 credentials not configured. Please set them in settings first")
	}
	
	oauth2Config.ClientID = cfg.OAuth2.ClientID
	oauth2Config.ClientSecret = cfg.OAuth2.ClientSecret
	// Generate state parameter for CSRF protection
	state, err := generateState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	// Channel to receive the authorization code
	codeChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	// Start HTTP server to handle OAuth callback
	mux := http.NewServeMux()
	server := &http.Server{Addr: ":8080", Handler: mux}
	
	// Add a root handler for debugging
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/callback" {
			// Handle callback
			if r.URL.Query().Get("state") != state {
				http.Error(w, "Invalid state parameter", http.StatusBadRequest)
				errorChan <- fmt.Errorf("invalid state parameter")
				return
			}

			code := r.URL.Query().Get("code")
			if code == "" {
				http.Error(w, "Authorization code not found", http.StatusBadRequest)
				errorChan <- fmt.Errorf("authorization code not found")
				return
			}

			// Redirect to success page in web settings
			http.Redirect(w, r, "http://localhost:8765/oauth-success", http.StatusTemporaryRedirect)

			codeChan <- code
		} else {
			// Handle other paths
			fmt.Fprintf(w, `
			<html>
			<head><title>MeetingBar OAuth Server</title></head>
			<body>
				<h2>MeetingBar OAuth Server</h2>
				<p>Server is running and waiting for OAuth callback...</p>
				<p>If you're seeing this, the server is working correctly.</p>
			</body>
			</html>
			`)
		}
	})

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errorChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	// Generate authorization URL
	authURL := oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	
	// Open browser to authorization URL
	if err := openBrowser(authURL); err != nil {
		log.Printf("Failed to open browser automatically: %v", err)
		fmt.Printf("Please open the following URL in your browser:\n%s\n", authURL)
	}

	// Wait for authorization code or timeout
	select {
	case code := <-codeChan:
		return exchangeCodeForAccount(ctx, code)
	case err := <-errorChan:
		return nil, err
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("authorization timeout")
	}
}

func exchangeCodeForAccount(ctx context.Context, code string) (*config.Account, error) {
	// Exchange authorization code for token
	token, err := oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Create OAuth2 client
	client := oauth2Config.Client(ctx, token)
	
	// Get user info to determine email
	userInfoService, err := oauth2api.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create user info service: %w", err)
	}
	
	userInfo, err := userInfoService.Userinfo.Get().Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Create account
	account := &config.Account{
		ID:      userInfo.Id,
		Email:   userInfo.Email,
		AddedAt: time.Now(),
	}

	// Store token securely
	if err := config.StoreToken(account.ID, token); err != nil {
		return nil, fmt.Errorf("failed to store token: %w", err)
	}

	return account, nil
}

func GetClientForAccount(ctx context.Context, accountID string) (*http.Client, error) {
	token, err := config.GetToken(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token for account %s: %w", accountID, err)
	}

	// Create token source that automatically refreshes
	tokenSource := oauth2Config.TokenSource(ctx, token)
	
	// Check if token needs refresh and update stored token
	refreshedToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	
	if refreshedToken.AccessToken != token.AccessToken {
		if err := config.StoreToken(accountID, refreshedToken); err != nil {
			log.Printf("Warning: failed to store refreshed token: %v", err)
		}
	}

	return oauth2.NewClient(ctx, tokenSource), nil
}

func generateState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	cmd = "xdg-open"
	args = []string{url}

	return exec.Command(cmd, args...).Start()
}