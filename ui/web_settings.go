package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os/exec"
	"time"

	"meetingbar/calendar"
	"meetingbar/config"
)

type WebSettingsManager struct {
	config          *config.Config
	calendarService *calendar.GoogleCalendarService
	notificationMgr *NotificationManager
	ctx             context.Context
	server          *http.Server
	port            int
}

type SettingsPageData struct {
	Config      *config.Config
	OAuth2Set   bool
	AccountsCount int
	CalendarsCount int
	NotificationStatus string
}

type APIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type AccountInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Avatar  string `json:"avatar"`
	AddedAt string `json:"addedAt"`
}

type AccountCalendarsInfo struct {
	Email         string        `json:"email"`
	Avatar        string        `json:"avatar"`
	CalendarCount int           `json:"calendarCount"`
	Calendars     []CalendarInfo `json:"calendars"`
}

type CalendarInfo struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Color       string `json:"color"`
	Selected    bool   `json:"selected"`
}

func NewWebSettingsManager(cfg *config.Config, ctx context.Context) *WebSettingsManager {
	return &WebSettingsManager{
		config:          cfg,
		calendarService: calendar.NewGoogleCalendarService(ctx),
		notificationMgr: NewNotificationManager(cfg),
		ctx:             ctx,
		port:            8765, // Different port from OAuth callback
	}
}

func (wsm *WebSettingsManager) ShowSettings() error {
	// Set up HTTP routes
	mux := http.NewServeMux()
	
	// Static pages
	mux.HandleFunc("/", wsm.handleHome)
	mux.HandleFunc("/oauth2", wsm.handleOAuth2Page)
	mux.HandleFunc("/accounts", wsm.handleAccountsPage)
	mux.HandleFunc("/calendars", wsm.handleCalendarsPage)
	mux.HandleFunc("/notifications", wsm.handleNotificationsPage)
	mux.HandleFunc("/general", wsm.handleGeneralPage)
	mux.HandleFunc("/oauth-success", wsm.handleOAuthSuccess)
	
	// API endpoints
	mux.HandleFunc("/api/oauth2", wsm.handleOAuth2API)
	mux.HandleFunc("/api/accounts", wsm.handleAccountsAPI)
	mux.HandleFunc("/api/calendars", wsm.handleCalendarsAPI)
	mux.HandleFunc("/api/notifications", wsm.handleNotificationsAPI)
	mux.HandleFunc("/api/general", wsm.handleGeneralAPI)
	mux.HandleFunc("/api/add-account", wsm.handleAddAccountAPI)
	mux.HandleFunc("/api/remove-account", wsm.handleRemoveAccountAPI)
	
	// Start server
	wsm.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", wsm.port),
		Handler: mux,
	}
	
	// Open browser
	url := fmt.Sprintf("http://localhost:%d", wsm.port)
	fmt.Printf("Opening settings in browser: %s\n", url)
	
	go func() {
		time.Sleep(500 * time.Millisecond)
		exec.Command("xdg-open", url).Start()
	}()
	
	// Start server (blocks until closed)
	fmt.Printf("Settings server running on %s\n", url)
	fmt.Println("Close this window when done with settings.")
	
	err := wsm.server.ListenAndServe()
	if err != http.ErrServerClosed {
		return fmt.Errorf("settings server error: %w", err)
	}
	
	return nil
}

func (wsm *WebSettingsManager) Close() {
	if wsm.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		wsm.server.Shutdown(ctx)
	}
}

func (wsm *WebSettingsManager) handleHome(w http.ResponseWriter, r *http.Request) {
	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>MeetingBar Settings</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 12px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        
        .header {
            background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
            color: white;
            padding: 30px;
            text-align: center;
        }
        
        .header h1 {
            font-size: 2.5rem;
            margin-bottom: 10px;
        }
        
        .header p {
            font-size: 1.1rem;
            opacity: 0.9;
        }
        
        .main-content {
            display: flex;
            min-height: 600px;
        }
        
        .sidebar {
            width: 300px;
            background: #f8fafc;
            border-right: 1px solid #e2e8f0;
            padding: 0;
        }
        
        .nav-item {
            display: block;
            padding: 20px 30px;
            text-decoration: none;
            color: #334155;
            border-bottom: 1px solid #e2e8f0;
            transition: all 0.3s ease;
            position: relative;
        }
        
        .nav-item:hover {
            background: #e2e8f0;
            color: #1e293b;
        }
        
        .nav-item.active {
            background: #3b82f6;
            color: white;
        }
        
        .nav-item .icon {
            font-size: 1.5rem;
            margin-right: 15px;
        }
        
        .nav-item .title {
            font-weight: 600;
            font-size: 1.1rem;
            display: block;
        }
        
        .nav-item .status {
            font-size: 0.9rem;
            opacity: 0.7;
            margin-top: 5px;
        }
        
        .content {
            flex: 1;
            padding: 40px;
        }
        
        .status-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        
        .status-card {
            background: #f8fafc;
            padding: 25px;
            border-radius: 8px;
            border-left: 4px solid #3b82f6;
        }
        
        .status-card.error {
            border-left-color: #ef4444;
        }
        
        .status-card.success {
            border-left-color: #10b981;
        }
        
        .status-card h3 {
            color: #1e293b;
            margin-bottom: 10px;
            display: flex;
            align-items: center;
        }
        
        .status-card .icon {
            margin-right: 10px;
            font-size: 1.3rem;
        }
        
        .setup-steps {
            background: white;
            border: 1px solid #e2e8f0;
            border-radius: 8px;
            padding: 30px;
        }
        
        .setup-steps h2 {
            color: #1e293b;
            margin-bottom: 20px;
            display: flex;
            align-items: center;
        }
        
        .setup-steps .icon {
            margin-right: 10px;
            font-size: 1.5rem;
        }
        
        .step {
            padding: 15px 0;
            border-bottom: 1px solid #f1f5f9;
        }
        
        .step:last-child {
            border-bottom: none;
        }
        
        .step-number {
            display: inline-flex;
            align-items: center;
            justify-content: center;
            width: 30px;
            height: 30px;
            background: #3b82f6;
            color: white;
            border-radius: 50%;
            font-weight: bold;
            margin-right: 15px;
        }
        
        .btn {
            display: inline-block;
            padding: 12px 24px;
            background: #3b82f6;
            color: white;
            text-decoration: none;
            border-radius: 6px;
            transition: background 0.3s ease;
            border: none;
            cursor: pointer;
            font-size: 1rem;
        }
        
        .btn:hover {
            background: #2563eb;
        }
        
        .btn-success {
            background: #10b981;
        }
        
        .btn-success:hover {
            background: #059669;
        }
        
        .footer {
            text-align: center;
            padding: 20px;
            background: #f8fafc;
            color: #64748b;
            border-top: 1px solid #e2e8f0;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>📅 MeetingBar Settings</h1>
            <p>Configure your calendar integration and meeting notifications</p>
        </div>
        
        <div class="main-content">
            <nav class="sidebar">
                <a href="/oauth2" class="nav-item">
                    <span class="icon">🔐</span>
                    <span class="title">OAuth2 Credentials</span>
                    <span class="status">{{if .OAuth2Set}}✅ Configured{{else}}❌ Not configured{{end}}</span>
                </a>
                <a href="/accounts" class="nav-item">
                    <span class="icon">👤</span>
                    <span class="title">Google Accounts</span>
                    <span class="status">{{.AccountsCount}} accounts</span>
                </a>
                <a href="/calendars" class="nav-item">
                    <span class="icon">📅</span>
                    <span class="title">Calendar Selection</span>
                    <span class="status">{{.CalendarsCount}} enabled</span>
                </a>
                <a href="/notifications" class="nav-item">
                    <span class="icon">🔔</span>
                    <span class="title">Notifications</span>
                    <span class="status">{{.NotificationStatus}}</span>
                </a>
                <a href="/general" class="nav-item">
                    <span class="icon">⚙️</span>
                    <span class="title">General Settings</span>
                    <span class="status">Refresh: {{.Config.RefreshInterval}}m</span>
                </a>
            </nav>
            
            <div class="content">
                <div class="status-grid">
                    <div class="status-card {{if .OAuth2Set}}success{{else}}error{{end}}">
                        <h3><span class="icon">🔐</span> OAuth2 Credentials</h3>
                        <p>{{if .OAuth2Set}}Ready to authenticate with Google{{else}}Required for Google Calendar access{{end}}</p>
                    </div>
                    
                    <div class="status-card {{if gt .AccountsCount 0}}success{{else}}error{{end}}">
                        <h3><span class="icon">👤</span> Google Accounts</h3>
                        <p>{{.AccountsCount}} account(s) configured</p>
                    </div>
                    
                    <div class="status-card {{if gt .CalendarsCount 0}}success{{else}}error{{end}}">
                        <h3><span class="icon">📅</span> Calendars</h3>
                        <p>{{.CalendarsCount}} calendar(s) enabled</p>
                    </div>
                    
                    <div class="status-card success">
                        <h3><span class="icon">🔔</span> Notifications</h3>
                        <p>{{.NotificationStatus}}</p>
                    </div>
                </div>
                
                <div class="setup-steps">
                    <h2><span class="icon">🚀</span> Quick Setup Guide</h2>
                    
                    <div class="step">
                        <span class="step-number">1</span>
                        <strong>Configure OAuth2 Credentials</strong>
                        <p>Set up Google Cloud Console credentials for calendar access.</p>
                        <a href="/oauth2" class="btn" style="margin-top: 10px;">Configure OAuth2</a>
                    </div>
                    
                    <div class="step">
                        <span class="step-number">2</span>
                        <strong>Add Google Account</strong>
                        <p>Connect your Google account to access calendar data.</p>
                        <a href="/accounts" class="btn" style="margin-top: 10px;">Manage Accounts</a>
                    </div>
                    
                    <div class="step">
                        <span class="step-number">3</span>
                        <strong>Select Calendars</strong>
                        <p>Choose which calendars to monitor for meetings.</p>
                        <a href="/calendars" class="btn" style="margin-top: 10px;">Select Calendars</a>
                    </div>
                    
                    <div class="step">
                        <span class="step-number">4</span>
                        <strong>Configure Notifications</strong>
                        <p>Set up meeting reminders and notification timing.</p>
                        <a href="/notifications" class="btn btn-success" style="margin-top: 10px;">Setup Notifications</a>
                    </div>
                </div>
            </div>
        </div>
        
        <div class="footer">
            <p>MeetingBar Settings • Close this window when finished</p>
        </div>
    </div>
</body>
</html>`

	data := SettingsPageData{
		Config:         wsm.config,
		OAuth2Set:      wsm.config.OAuth2.ClientID != "" && wsm.config.OAuth2.ClientSecret != "",
		AccountsCount:  len(wsm.config.Accounts),
		CalendarsCount: len(wsm.config.EnabledCalendars),
		NotificationStatus: wsm.getNotificationStatus(),
	}

	t, err := template.New("home").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	t.Execute(w, data)
}

func (wsm *WebSettingsManager) handleOAuth2Page(w http.ResponseWriter, r *http.Request) {
	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>OAuth2 Credentials - MeetingBar</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        
        .container {
            max-width: 800px;
            margin: 0 auto;
            background: white;
            border-radius: 12px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        
        .header {
            background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
            color: white;
            padding: 30px;
            text-align: center;
        }
        
        .content {
            padding: 40px;
        }
        
        .form-group {
            margin-bottom: 25px;
        }
        
        .form-group label {
            display: block;
            margin-bottom: 8px;
            font-weight: 600;
            color: #374151;
        }
        
        .form-group input {
            width: 100%;
            padding: 12px;
            border: 2px solid #e5e7eb;
            border-radius: 6px;
            font-size: 1rem;
            transition: border-color 0.3s ease;
        }
        
        .form-group input:focus {
            outline: none;
            border-color: #3b82f6;
        }
        
        .btn {
            display: inline-block;
            padding: 12px 24px;
            background: #3b82f6;
            color: white;
            text-decoration: none;
            border-radius: 6px;
            transition: background 0.3s ease;
            border: none;
            cursor: pointer;
            font-size: 1rem;
            margin-right: 10px;
        }
        
        .btn:hover {
            background: #2563eb;
        }
        
        .btn-danger {
            background: #ef4444;
        }
        
        .btn-danger:hover {
            background: #dc2626;
        }
        
        .instructions {
            background: #f0f9ff;
            border: 1px solid #0ea5e9;
            border-radius: 8px;
            padding: 20px;
            margin-bottom: 30px;
        }
        
        .instructions h3 {
            color: #0c4a6e;
            margin-bottom: 15px;
        }
        
        .instructions ol {
            color: #0c4a6e;
            padding-left: 20px;
        }
        
        .instructions li {
            margin-bottom: 8px;
        }
        
        .back-link {
            display: inline-block;
            margin-bottom: 20px;
            color: #3b82f6;
            text-decoration: none;
        }
        
        .back-link:hover {
            text-decoration: underline;
        }
        
        .status {
            padding: 15px;
            border-radius: 6px;
            margin-bottom: 20px;
        }
        
        .status.success {
            background: #f0fdf4;
            border: 1px solid #16a34a;
            color: #15803d;
        }
        
        .status.error {
            background: #fef2f2;
            border: 1px solid #ef4444;
            color: #dc2626;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔐 OAuth2 Credentials</h1>
            <p>Configure Google Calendar API access</p>
        </div>
        
        <div class="content">
            <a href="/" class="back-link">← Back to Settings</a>
            
            <div class="instructions">
                <h3>📋 Setup Instructions</h3>
                <ol>
                    <li>Go to <strong>Google Cloud Console</strong> (console.cloud.google.com)</li>
                    <li>Create a new project or select an existing one</li>
                    <li>Enable the <strong>Google Calendar API</strong></li>
                    <li>Create <strong>OAuth 2.0 Client IDs</strong>:
                        <ul style="margin-top: 5px;">
                            <li>Application type: <strong>Desktop application</strong></li>
                            <li>Authorized redirect URIs: <strong>http://localhost:8080/callback</strong></li>
                        </ul>
                    </li>
                    <li>Copy the Client ID and Client Secret to the form below</li>
                </ol>
            </div>
            
            {{if .OAuth2Set}}
            <div class="status success">
                ✅ OAuth2 credentials are configured! Client ID: {{.ClientIDPreview}}
            </div>
            {{else}}
            <div class="status error">
                ❌ OAuth2 credentials not configured. Please enter them below.
            </div>
            {{end}}
            
            <form id="oauth2Form">
                <div class="form-group">
                    <label for="clientId">Google OAuth2 Client ID:</label>
                    <input type="text" id="clientId" name="clientId" placeholder="your-client-id.googleusercontent.com" value="{{.Config.OAuth2.ClientID}}">
                </div>
                
                <div class="form-group">
                    <label for="clientSecret">Google OAuth2 Client Secret:</label>
                    <input type="password" id="clientSecret" name="clientSecret" placeholder="Your client secret" value="{{.Config.OAuth2.ClientSecret}}">
                </div>
                
                <button type="submit" class="btn">💾 Save Credentials</button>
                <button type="button" class="btn btn-danger" onclick="clearCredentials()">🗑️ Clear Credentials</button>
            </form>
        </div>
    </div>
    
    <script>
        document.getElementById('oauth2Form').addEventListener('submit', async (e) => {
            e.preventDefault();
            
            const formData = new FormData(e.target);
            const data = {
                clientId: formData.get('clientId'),
                clientSecret: formData.get('clientSecret')
            };
            
            try {
                const response = await fetch('/api/oauth2', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(data)
                });
                
                const result = await response.json();
                
                if (result.success) {
                    alert('✅ OAuth2 credentials saved successfully!');
                    location.reload();
                } else {
                    alert('❌ Error: ' + result.message);
                }
            } catch (error) {
                alert('❌ Error saving credentials: ' + error.message);
            }
        });
        
        async function clearCredentials() {
            if (!confirm('Are you sure you want to clear OAuth2 credentials?')) {
                return;
            }
            
            try {
                const response = await fetch('/api/oauth2', {
                    method: 'DELETE'
                });
                
                const result = await response.json();
                
                if (result.success) {
                    alert('✅ OAuth2 credentials cleared!');
                    location.reload();
                } else {
                    alert('❌ Error: ' + result.message);
                }
            } catch (error) {
                alert('❌ Error clearing credentials: ' + error.message);
            }
        }
    </script>
</body>
</html>`

	data := struct {
		Config           *config.Config
		OAuth2Set        bool
		ClientIDPreview  string
	}{
		Config:    wsm.config,
		OAuth2Set: wsm.config.OAuth2.ClientID != "" && wsm.config.OAuth2.ClientSecret != "",
		ClientIDPreview: wsm.getClientIDPreview(),
	}

	t, err := template.New("oauth2").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	t.Execute(w, data)
}

func (wsm *WebSettingsManager) handleOAuth2API(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "POST":
		var data struct {
			ClientID     string `json:"clientId"`
			ClientSecret string `json:"clientSecret"`
		}

		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Invalid JSON"})
			return
		}

		if data.ClientID == "" || data.ClientSecret == "" {
			json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Both Client ID and Client Secret are required"})
			return
		}

		wsm.config.OAuth2.ClientID = data.ClientID
		wsm.config.OAuth2.ClientSecret = data.ClientSecret

		if err := wsm.config.Save(); err != nil {
			json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Failed to save configuration"})
			return
		}

		json.NewEncoder(w).Encode(APIResponse{Success: true, Message: "OAuth2 credentials saved successfully"})

	case "DELETE":
		wsm.config.OAuth2.ClientID = ""
		wsm.config.OAuth2.ClientSecret = ""

		if err := wsm.config.Save(); err != nil {
			json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Failed to save configuration"})
			return
		}

		json.NewEncoder(w).Encode(APIResponse{Success: true, Message: "OAuth2 credentials cleared"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (wsm *WebSettingsManager) handleAccountsPage(w http.ResponseWriter, r *http.Request) {
	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Google Accounts - MeetingBar</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        
        .container {
            max-width: 900px;
            margin: 0 auto;
            background: white;
            border-radius: 12px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        
        .header {
            background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
            color: white;
            padding: 30px;
            text-align: center;
        }
        
        .content {
            padding: 40px;
        }
        
        .back-link {
            display: inline-block;
            margin-bottom: 20px;
            color: #3b82f6;
            text-decoration: none;
        }
        
        .back-link:hover {
            text-decoration: underline;
        }
        
        .accounts-grid {
            display: grid;
            gap: 20px;
            margin-bottom: 30px;
        }
        
        .account-card {
            background: #f8fafc;
            border: 1px solid #e2e8f0;
            border-radius: 8px;
            padding: 25px;
            display: flex;
            align-items: center;
            justify-content: space-between;
        }
        
        .account-info {
            display: flex;
            align-items: center;
        }
        
        .account-avatar {
            width: 48px;
            height: 48px;
            border-radius: 50%;
            background: #3b82f6;
            color: white;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 1.5rem;
            margin-right: 15px;
        }
        
        .account-details h3 {
            color: #1e293b;
            margin-bottom: 5px;
        }
        
        .account-details p {
            color: #64748b;
            font-size: 0.9rem;
        }
        
        .account-actions {
            display: flex;
            gap: 10px;
        }
        
        .btn {
            display: inline-block;
            padding: 10px 20px;
            background: #3b82f6;
            color: white;
            text-decoration: none;
            border-radius: 6px;
            transition: background 0.3s ease;
            border: none;
            cursor: pointer;
            font-size: 0.9rem;
        }
        
        .btn:hover {
            background: #2563eb;
        }
        
        .btn-danger {
            background: #ef4444;
        }
        
        .btn-danger:hover {
            background: #dc2626;
        }
        
        .btn-success {
            background: #10b981;
        }
        
        .btn-success:hover {
            background: #059669;
        }
        
        .add-account {
            text-align: center;
            padding: 40px;
            border: 2px dashed #cbd5e0;
            border-radius: 8px;
            margin-bottom: 30px;
        }
        
        .add-account h3 {
            color: #4a5568;
            margin-bottom: 15px;
        }
        
        .add-account p {
            color: #718096;
            margin-bottom: 20px;
        }
        
        .instructions {
            background: #f0f9ff;
            border: 1px solid #0ea5e9;
            border-radius: 8px;
            padding: 20px;
            margin-bottom: 20px;
        }
        
        .instructions h4 {
            color: #0c4a6e;
            margin-bottom: 10px;
        }
        
        .instructions p {
            color: #0c4a6e;
            font-size: 0.9rem;
        }
        
        .warning {
            background: #fef3c7;
            border: 1px solid #f59e0b;
            border-radius: 8px;
            padding: 15px;
            margin-bottom: 20px;
        }
        
        .warning p {
            color: #92400e;
            font-size: 0.9rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>👤 Google Accounts</h1>
            <p>Manage your Google Calendar accounts</p>
        </div>
        
        <div class="content">
            <a href="/" class="back-link">← Back to Settings</a>
            
            {{if not .OAuth2Set}}
            <div class="warning">
                <p>⚠️ You need to configure OAuth2 credentials first before adding accounts.</p>
            </div>
            {{end}}
            
            {{if .Accounts}}
            <div class="accounts-grid">
                {{range .Accounts}}
                <div class="account-card">
                    <div class="account-info">
                        <div class="account-avatar">{{.Avatar}}</div>
                        <div class="account-details">
                            <h3>{{.Email}}</h3>
                            <p>Added: {{.AddedAt}}</p>
                        </div>
                    </div>
                    <div class="account-actions">
                        <button class="btn" onclick="refreshAccount('{{.ID}}')">🔄 Refresh</button>
                        <button class="btn btn-danger" onclick="removeAccount('{{.ID}}')">🗑️ Remove</button>
                    </div>
                </div>
                {{end}}
            </div>
            {{end}}
            
            <div class="add-account">
                <h3>Add New Google Account</h3>
                <p>Connect another Google account to access more calendars</p>
                
                {{if .OAuth2Set}}
                <button class="btn btn-success" onclick="addAccount()">+ Add Google Account</button>
                {{else}}
                <a href="/oauth2" class="btn">Configure OAuth2 First</a>
                {{end}}
            </div>
            
            {{if .OAuth2Set}}
            <div class="instructions">
                <h4>📋 How it works:</h4>
                <p>When you click "Add Google Account", you'll be redirected to Google's login page. After signing in and granting permissions, your account will be automatically added to MeetingBar. This may take a few moments to complete.</p>
            </div>
            {{end}}
        </div>
    </div>
    
    <script>
        async function addAccount() {
            try {
                // Show loading state
                document.querySelector('button[onclick="addAccount()"]').textContent = 'Starting authentication...';
                document.querySelector('button[onclick="addAccount()"]').disabled = true;
                
                const response = await fetch('/api/add-account', {
                    method: 'POST'
                });
                
                const result = await response.json();
                
                if (result.success && result.data && result.data.authUrl) {
                    alert('ℹ️ You will be redirected to Google for authentication. After completing the process, please refresh this page to see your new account.');
                    // Open Google OAuth URL in current window
                    window.location.href = result.data.authUrl;
                } else {
                    alert('❌ Error: ' + (result.message || 'Failed to start authentication'));
                    // Reset button
                    document.querySelector('button[onclick="addAccount()"]').textContent = '+ Add Google Account';
                    document.querySelector('button[onclick="addAccount()"]').disabled = false;
                }
            } catch (error) {
                alert('❌ Error adding account: ' + error.message);
                // Reset button
                document.querySelector('button[onclick="addAccount()"]').textContent = '+ Add Google Account';
                document.querySelector('button[onclick="addAccount()"]').disabled = false;
            }
        }
        
        async function removeAccount(accountId) {
            if (!confirm('Are you sure you want to remove this account?')) {
                return;
            }
            
            try {
                const response = await fetch('/api/remove-account', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ accountId: accountId })
                });
                
                const result = await response.json();
                
                if (result.success) {
                    alert('✅ Account removed successfully!');
                    location.reload();
                } else {
                    alert('❌ Error: ' + result.message);
                }
            } catch (error) {
                alert('❌ Error removing account: ' + error.message);
            }
        }
        
        async function refreshAccount(accountId) {
            try {
                const response = await fetch('/api/accounts', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ action: 'refresh', accountId: accountId })
                });
                
                const result = await response.json();
                
                if (result.success) {
                    alert('✅ Account refreshed successfully!');
                    location.reload();
                } else {
                    alert('❌ Error: ' + result.message);
                }
            } catch (error) {
                alert('❌ Error refreshing account: ' + error.message);
            }
        }
    </script>
</body>
</html>`

	data := struct {
		Config    *config.Config
		OAuth2Set bool
		Accounts  []AccountInfo
	}{
		Config:    wsm.config,
		OAuth2Set: wsm.config.OAuth2.ClientID != "" && wsm.config.OAuth2.ClientSecret != "",
		Accounts:  wsm.getAccountsInfo(),
	}

	t, err := template.New("accounts").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	t.Execute(w, data)
}

func (wsm *WebSettingsManager) handleCalendarsPage(w http.ResponseWriter, r *http.Request) {
	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Calendar Selection - MeetingBar</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        
        .container {
            max-width: 900px;
            margin: 0 auto;
            background: white;
            border-radius: 12px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        
        .header {
            background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
            color: white;
            padding: 30px;
            text-align: center;
        }
        
        .content {
            padding: 40px;
        }
        
        .back-link {
            display: inline-block;
            margin-bottom: 20px;
            color: #3b82f6;
            text-decoration: none;
        }
        
        .back-link:hover {
            text-decoration: underline;
        }
        
        .account-section {
            margin-bottom: 40px;
        }
        
        .account-header {
            background: #f8fafc;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 20px;
            display: flex;
            align-items: center;
        }
        
        .account-avatar {
            width: 40px;
            height: 40px;
            border-radius: 50%;
            background: #3b82f6;
            color: white;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 1.2rem;
            margin-right: 15px;
        }
        
        .account-info h3 {
            color: #1e293b;
            margin-bottom: 5px;
        }
        
        .account-info p {
            color: #64748b;
            font-size: 0.9rem;
        }
        
        .calendars-grid {
            display: grid;
            gap: 15px;
        }
        
        .calendar-item {
            background: white;
            border: 2px solid #e2e8f0;
            border-radius: 8px;
            padding: 20px;
            display: flex;
            align-items: center;
            transition: all 0.3s ease;
        }
        
        .calendar-item:hover {
            border-color: #cbd5e0;
        }
        
        .calendar-item.selected {
            border-color: #3b82f6;
            background: #f0f9ff;
        }
        
        .calendar-checkbox {
            width: 20px;
            height: 20px;
            margin-right: 15px;
            cursor: pointer;
        }
        
        .calendar-info {
            flex: 1;
        }
        
        .calendar-info h4 {
            color: #1e293b;
            margin-bottom: 5px;
        }
        
        .calendar-info p {
            color: #64748b;
            font-size: 0.9rem;
        }
        
        .calendar-color {
            width: 20px;
            height: 20px;
            border-radius: 50%;
            margin-left: 15px;
        }
        
        .btn {
            display: inline-block;
            padding: 12px 24px;
            background: #3b82f6;
            color: white;
            text-decoration: none;
            border-radius: 6px;
            transition: background 0.3s ease;
            border: none;
            cursor: pointer;
            font-size: 1rem;
            margin-right: 10px;
        }
        
        .btn:hover {
            background: #2563eb;
        }
        
        .btn-success {
            background: #10b981;
        }
        
        .btn-success:hover {
            background: #059669;
        }
        
        .actions {
            text-align: center;
            margin-top: 30px;
            padding-top: 30px;
            border-top: 1px solid #e2e8f0;
        }
        
        .warning {
            background: #fef3c7;
            border: 1px solid #f59e0b;
            border-radius: 8px;
            padding: 15px;
            margin-bottom: 20px;
        }
        
        .warning p {
            color: #92400e;
            font-size: 0.9rem;
        }
        
        .info {
            background: #f0f9ff;
            border: 1px solid #0ea5e9;
            border-radius: 8px;
            padding: 15px;
            margin-bottom: 20px;
        }
        
        .info p {
            color: #0c4a6e;
            font-size: 0.9rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>📅 Calendar Selection</h1>
            <p>Choose which calendars to monitor for meetings</p>
        </div>
        
        <div class="content">
            <a href="/" class="back-link">← Back to Settings</a>
            
            {{if not .HasAccounts}}
            <div class="warning">
                <p>⚠️ You need to add Google accounts first before selecting calendars.</p>
            </div>
            {{end}}
            
            {{if .HasAccounts}}
            <div class="info">
                <p>📋 Select the calendars you want MeetingBar to monitor. Only meetings from selected calendars will appear in your tray.</p>
            </div>
            
            {{range .AccountCalendars}}
            <div class="account-section">
                <div class="account-header">
                    <div class="account-avatar">{{.Avatar}}</div>
                    <div class="account-info">
                        <h3>{{.Email}}</h3>
                        <p>{{.CalendarCount}} calendars available</p>
                    </div>
                </div>
                
                <div class="calendars-grid">
                    {{range .Calendars}}
                    <div class="calendar-item {{if .Selected}}selected{{end}}" onclick="toggleCalendar('{{.ID}}', this)">
                        <input type="checkbox" class="calendar-checkbox" 
                               id="cal_{{.ID}}" 
                               {{if .Selected}}checked{{end}}
                               onchange="toggleCalendar('{{.ID}}', this.parentElement)">
                        <div class="calendar-info">
                            <h4>{{.Title}}</h4>
                            <p>{{.Description}}</p>
                        </div>
                        <div class="calendar-color" style="background-color: {{.Color}}"></div>
                    </div>
                    {{end}}
                </div>
            </div>
            {{end}}
            
            <div class="actions">
                <button class="btn btn-success" onclick="saveCalendarSelection()">💾 Save Selection</button>
                <button class="btn" onclick="selectAll()">✅ Select All</button>
                <button class="btn" onclick="selectNone()">❌ Select None</button>
            </div>
            {{else}}
            <div style="text-align: center; padding: 40px;">
                <a href="/accounts" class="btn">Add Google Accounts First</a>
            </div>
            {{end}}
        </div>
    </div>
    
    <script>
        let selectedCalendars = new Set();
        
        // Initialize selected calendars
        document.querySelectorAll('.calendar-checkbox:checked').forEach(checkbox => {
            selectedCalendars.add(checkbox.id.replace('cal_', ''));
        });
        
        function toggleCalendar(calendarId, element) {
            const checkbox = element.querySelector('.calendar-checkbox');
            const isChecked = checkbox.checked;
            
            if (isChecked) {
                selectedCalendars.add(calendarId);
                element.classList.add('selected');
            } else {
                selectedCalendars.delete(calendarId);
                element.classList.remove('selected');
            }
        }
        
        function selectAll() {
            document.querySelectorAll('.calendar-checkbox').forEach(checkbox => {
                checkbox.checked = true;
                const calendarId = checkbox.id.replace('cal_', '');
                selectedCalendars.add(calendarId);
                checkbox.parentElement.classList.add('selected');
            });
        }
        
        function selectNone() {
            document.querySelectorAll('.calendar-checkbox').forEach(checkbox => {
                checkbox.checked = false;
                const calendarId = checkbox.id.replace('cal_', '');
                selectedCalendars.delete(calendarId);
                checkbox.parentElement.classList.remove('selected');
            });
        }
        
        async function saveCalendarSelection() {
            try {
                const response = await fetch('/api/calendars', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ 
                        action: 'save',
                        selectedCalendars: Array.from(selectedCalendars)
                    })
                });
                
                const result = await response.json();
                
                if (result.success) {
                    alert('✅ Calendar selection saved successfully!');
                } else {
                    alert('❌ Error: ' + result.message);
                }
            } catch (error) {
                alert('❌ Error saving calendar selection: ' + error.message);
            }
        }
    </script>
</body>
</html>`

	data := struct {
		Config           *config.Config
		HasAccounts      bool
		AccountCalendars []AccountCalendarsInfo
	}{
		Config:           wsm.config,
		HasAccounts:      len(wsm.config.Accounts) > 0,
		AccountCalendars: wsm.getAccountCalendarsInfo(),
	}

	t, err := template.New("calendars").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	t.Execute(w, data)
}

func (wsm *WebSettingsManager) handleNotificationsPage(w http.ResponseWriter, r *http.Request) {
	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Notifications - MeetingBar</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        
        .container {
            max-width: 800px;
            margin: 0 auto;
            background: white;
            border-radius: 12px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        
        .header {
            background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
            color: white;
            padding: 30px;
            text-align: center;
        }
        
        .content {
            padding: 40px;
        }
        
        .back-link {
            display: inline-block;
            margin-bottom: 20px;
            color: #3b82f6;
            text-decoration: none;
        }
        
        .back-link:hover {
            text-decoration: underline;
        }
        
        .settings-section {
            background: #f8fafc;
            border: 1px solid #e2e8f0;
            border-radius: 8px;
            padding: 30px;
            margin-bottom: 30px;
        }
        
        .settings-section h3 {
            color: #1e293b;
            margin-bottom: 20px;
            display: flex;
            align-items: center;
        }
        
        .settings-section .icon {
            margin-right: 10px;
            font-size: 1.3rem;
        }
        
        .setting-item {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 20px 0;
            border-bottom: 1px solid #e2e8f0;
        }
        
        .setting-item:last-child {
            border-bottom: none;
        }
        
        .setting-info {
            flex: 1;
        }
        
        .setting-info h4 {
            color: #1e293b;
            margin-bottom: 5px;
        }
        
        .setting-info p {
            color: #64748b;
            font-size: 0.9rem;
        }
        
        .setting-control {
            margin-left: 20px;
        }
        
        .toggle {
            position: relative;
            display: inline-block;
            width: 60px;
            height: 34px;
        }
        
        .toggle input {
            opacity: 0;
            width: 0;
            height: 0;
        }
        
        .slider {
            position: absolute;
            cursor: pointer;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background-color: #ccc;
            transition: .4s;
            border-radius: 34px;
        }
        
        .slider:before {
            position: absolute;
            content: "";
            height: 26px;
            width: 26px;
            left: 4px;
            bottom: 4px;
            background-color: white;
            transition: .4s;
            border-radius: 50%;
        }
        
        input:checked + .slider {
            background-color: #3b82f6;
        }
        
        input:checked + .slider:before {
            transform: translateX(26px);
        }
        
        .form-group {
            margin-bottom: 20px;
        }
        
        .form-group label {
            display: block;
            margin-bottom: 8px;
            font-weight: 600;
            color: #374151;
        }
        
        .form-group select,
        .form-group input {
            width: 100%;
            padding: 12px;
            border: 2px solid #e5e7eb;
            border-radius: 6px;
            font-size: 1rem;
            transition: border-color 0.3s ease;
        }
        
        .form-group select:focus,
        .form-group input:focus {
            outline: none;
            border-color: #3b82f6;
        }
        
        .btn {
            display: inline-block;
            padding: 12px 24px;
            background: #3b82f6;
            color: white;
            text-decoration: none;
            border-radius: 6px;
            transition: background 0.3s ease;
            border: none;
            cursor: pointer;
            font-size: 1rem;
            margin-right: 10px;
        }
        
        .btn:hover {
            background: #2563eb;
        }
        
        .btn-success {
            background: #10b981;
        }
        
        .btn-success:hover {
            background: #059669;
        }
        
        .actions {
            text-align: center;
            margin-top: 30px;
            padding-top: 30px;
            border-top: 1px solid #e2e8f0;
        }
        
        .preview {
            background: #f0f9ff;
            border: 1px solid #0ea5e9;
            border-radius: 8px;
            padding: 15px;
            margin-top: 15px;
        }
        
        .preview p {
            color: #0c4a6e;
            font-size: 0.9rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔔 Notifications</h1>
            <p>Configure meeting reminders and alerts</p>
        </div>
        
        <div class="content">
            <a href="/" class="back-link">← Back to Settings</a>
            
            <div class="settings-section">
                <h3><span class="icon">🔔</span> Notification Settings</h3>
                
                <div class="setting-item">
                    <div class="setting-info">
                        <h4>Enable Notifications</h4>
                        <p>Show desktop notifications for upcoming meetings</p>
                    </div>
                    <div class="setting-control">
                        <label class="toggle">
                            <input type="checkbox" id="enableNotifications" {{if .Config.EnableNotifications}}checked{{end}}>
                            <span class="slider"></span>
                        </label>
                    </div>
                </div>
                
                <div class="setting-item">
                    <div class="setting-info">
                        <h4>Notification Timing</h4>
                        <p>How many minutes before the meeting to show notifications</p>
                    </div>
                    <div class="setting-control">
                        <div class="form-group" style="margin: 0; width: 120px;">
                            <select id="notificationTime">
                                <option value="1" {{if eq .Config.NotificationTime 1}}selected{{end}}>1 minute</option>
                                <option value="2" {{if eq .Config.NotificationTime 2}}selected{{end}}>2 minutes</option>
                                <option value="5" {{if eq .Config.NotificationTime 5}}selected{{end}}>5 minutes</option>
                                <option value="10" {{if eq .Config.NotificationTime 10}}selected{{end}}>10 minutes</option>
                                <option value="15" {{if eq .Config.NotificationTime 15}}selected{{end}}>15 minutes</option>
                                <option value="30" {{if eq .Config.NotificationTime 30}}selected{{end}}>30 minutes</option>
                            </select>
                        </div>
                    </div>
                </div>
                
                <div class="setting-item">
                    <div class="setting-info">
                        <h4>Show Meeting Links</h4>
                        <p>Include join links in notification messages</p>
                    </div>
                    <div class="setting-control">
                        <label class="toggle">
                            <input type="checkbox" id="showMeetingLinks" {{if .Config.ShowMeetingLinks}}checked{{end}}>
                            <span class="slider"></span>
                        </label>
                    </div>
                </div>
                
                <div class="setting-item">
                    <div class="setting-info">
                        <h4>Persistent Notifications</h4>
                        <p>Keep notifications visible until dismissed</p>
                    </div>
                    <div class="setting-control">
                        <label class="toggle">
                            <input type="checkbox" id="persistentNotifications" {{if .Config.PersistentNotifications}}checked{{end}}>
                            <span class="slider"></span>
                        </label>
                    </div>
                </div>
            </div>
            
            <div class="settings-section">
                <h3><span class="icon">🔊</span> Sound Settings</h3>
                
                <div class="setting-item">
                    <div class="setting-info">
                        <h4>Notification Sound</h4>
                        <p>Play a sound when showing meeting notifications</p>
                    </div>
                    <div class="setting-control">
                        <label class="toggle">
                            <input type="checkbox" id="notificationSound" {{if .Config.NotificationSound}}checked{{end}}>
                            <span class="slider"></span>
                        </label>
                    </div>
                </div>
            </div>
            
            <div class="preview">
                <p><strong>Preview:</strong> {{.PreviewText}}</p>
            </div>
            
            <div class="actions">
                <button class="btn btn-success" onclick="saveNotificationSettings()">💾 Save Settings</button>
                <button class="btn" onclick="testNotification()">🗏 Test Notification</button>
            </div>
        </div>
    </div>
    
    <script>
        function updatePreview() {
            const enabled = document.getElementById('enableNotifications').checked;
            const time = document.getElementById('notificationTime').value;
            const showLinks = document.getElementById('showMeetingLinks').checked;
            const persistent = document.getElementById('persistentNotifications').checked;
            const sound = document.getElementById('notificationSound').checked;
            
            let preview = "Notifications: ";
            if (enabled) {
                preview += "Enabled, " + time + " minutes before meetings";
                if (showLinks) preview += ", with meeting links";
                if (persistent) preview += ", persistent";
                if (sound) preview += ", with sound";
            } else {
                preview += "Disabled";
            }
            
            document.querySelector('.preview p').innerHTML = "<strong>Preview:</strong> " + preview;
        }
        
        // Update preview when settings change
        document.querySelectorAll('input, select').forEach(element => {
            element.addEventListener('change', updatePreview);
        });
        
        async function saveNotificationSettings() {
            const settings = {
                enableNotifications: document.getElementById('enableNotifications').checked,
                notificationTime: parseInt(document.getElementById('notificationTime').value),
                showMeetingLinks: document.getElementById('showMeetingLinks').checked,
                persistentNotifications: document.getElementById('persistentNotifications').checked,
                notificationSound: document.getElementById('notificationSound').checked
            };
            
            try {
                const response = await fetch('/api/notifications', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ action: 'save', settings: settings })
                });
                
                const result = await response.json();
                
                if (result.success) {
                    alert('✅ Notification settings saved successfully!');
                } else {
                    alert('❌ Error: ' + result.message);
                }
            } catch (error) {
                alert('❌ Error saving settings: ' + error.message);
            }
        }
        
        async function testNotification() {
            try {
                const response = await fetch('/api/notifications', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ action: 'test' })
                });
                
                const result = await response.json();
                
                if (result.success) {
                    alert('✅ Test notification sent!');
                } else {
                    alert('❌ Error: ' + result.message);
                }
            } catch (error) {
                alert('❌ Error sending test notification: ' + error.message);
            }
        }
        
        // Initialize preview
        updatePreview();
    </script>
</body>
</html>`

	data := struct {
		Config      *config.Config
		PreviewText string
	}{
		Config:      wsm.config,
		PreviewText: wsm.getNotificationPreview(),
	}

	t, err := template.New("notifications").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	t.Execute(w, data)
}

func (wsm *WebSettingsManager) handleGeneralPage(w http.ResponseWriter, r *http.Request) {
	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>General Settings - MeetingBar</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        
        .container {
            max-width: 800px;
            margin: 0 auto;
            background: white;
            border-radius: 12px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        
        .header {
            background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
            color: white;
            padding: 30px;
            text-align: center;
        }
        
        .content {
            padding: 40px;
        }
        
        .back-link {
            display: inline-block;
            margin-bottom: 20px;
            color: #3b82f6;
            text-decoration: none;
        }
        
        .back-link:hover {
            text-decoration: underline;
        }
        
        .settings-section {
            background: #f8fafc;
            border: 1px solid #e2e8f0;
            border-radius: 8px;
            padding: 30px;
            margin-bottom: 30px;
        }
        
        .settings-section h3 {
            color: #1e293b;
            margin-bottom: 20px;
            display: flex;
            align-items: center;
        }
        
        .settings-section .icon {
            margin-right: 10px;
            font-size: 1.3rem;
        }
        
        .setting-item {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 20px 0;
            border-bottom: 1px solid #e2e8f0;
        }
        
        .setting-item:last-child {
            border-bottom: none;
        }
        
        .setting-info {
            flex: 1;
        }
        
        .setting-info h4 {
            color: #1e293b;
            margin-bottom: 5px;
        }
        
        .setting-info p {
            color: #64748b;
            font-size: 0.9rem;
        }
        
        .setting-control {
            margin-left: 20px;
        }
        
        .form-group {
            margin-bottom: 20px;
        }
        
        .form-group label {
            display: block;
            margin-bottom: 8px;
            font-weight: 600;
            color: #374151;
        }
        
        .form-group select,
        .form-group input {
            width: 100%;
            padding: 12px;
            border: 2px solid #e5e7eb;
            border-radius: 6px;
            font-size: 1rem;
            transition: border-color 0.3s ease;
        }
        
        .form-group select:focus,
        .form-group input:focus {
            outline: none;
            border-color: #3b82f6;
        }
        
        .toggle {
            position: relative;
            display: inline-block;
            width: 60px;
            height: 34px;
        }
        
        .toggle input {
            opacity: 0;
            width: 0;
            height: 0;
        }
        
        .slider {
            position: absolute;
            cursor: pointer;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background-color: #ccc;
            transition: .4s;
            border-radius: 34px;
        }
        
        .slider:before {
            position: absolute;
            content: "";
            height: 26px;
            width: 26px;
            left: 4px;
            bottom: 4px;
            background-color: white;
            transition: .4s;
            border-radius: 50%;
        }
        
        input:checked + .slider {
            background-color: #3b82f6;
        }
        
        input:checked + .slider:before {
            transform: translateX(26px);
        }
        
        .btn {
            display: inline-block;
            padding: 12px 24px;
            background: #3b82f6;
            color: white;
            text-decoration: none;
            border-radius: 6px;
            transition: background 0.3s ease;
            border: none;
            cursor: pointer;
            font-size: 1rem;
            margin-right: 10px;
        }
        
        .btn:hover {
            background: #2563eb;
        }
        
        .btn-success {
            background: #10b981;
        }
        
        .btn-success:hover {
            background: #059669;
        }
        
        .btn-danger {
            background: #ef4444;
        }
        
        .btn-danger:hover {
            background: #dc2626;
        }
        
        .actions {
            text-align: center;
            margin-top: 30px;
            padding-top: 30px;
            border-top: 1px solid #e2e8f0;
        }
        
        .config-viewer {
            background: #1e293b;
            color: #e2e8f0;
            padding: 20px;
            border-radius: 8px;
            font-family: 'Monaco', 'Menlo', monospace;
            font-size: 0.9rem;
            white-space: pre-wrap;
            max-height: 400px;
            overflow-y: auto;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>⚙️ General Settings</h1>
            <p>Configure application behavior and preferences</p>
        </div>
        
        <div class="content">
            <a href="/" class="back-link">← Back to Settings</a>
            
            <div class="settings-section">
                <h3><span class="icon">🔄</span> Refresh Settings</h3>
                
                <div class="setting-item">
                    <div class="setting-info">
                        <h4>Calendar Refresh Interval</h4>
                        <p>How often to check for new meetings and updates</p>
                    </div>
                    <div class="setting-control">
                        <div class="form-group" style="margin: 0; width: 150px;">
                            <select id="refreshInterval">
                                <option value="1" {{if eq .Config.RefreshInterval 1}}selected{{end}}>1 minute</option>
                                <option value="2" {{if eq .Config.RefreshInterval 2}}selected{{end}}>2 minutes</option>
                                <option value="5" {{if eq .Config.RefreshInterval 5}}selected{{end}}>5 minutes</option>
                                <option value="10" {{if eq .Config.RefreshInterval 10}}selected{{end}}>10 minutes</option>
                                <option value="15" {{if eq .Config.RefreshInterval 15}}selected{{end}}>15 minutes</option>
                                <option value="30" {{if eq .Config.RefreshInterval 30}}selected{{end}}>30 minutes</option>
                            </select>
                        </div>
                    </div>
                </div>
            </div>
            
            <div class="settings-section">
                <h3><span class="icon">📺</span> Display Settings</h3>
                
                <div class="setting-item">
                    <div class="setting-info">
                        <h4>Show Meeting Duration in Tray</h4>
                        <p>Display meeting duration in the system tray title</p>
                    </div>
                    <div class="setting-control">
                        <label class="toggle">
                            <input type="checkbox" id="showDuration" {{if .Config.ShowDuration}}checked{{end}}>
                            <span class="slider"></span>
                        </label>
                    </div>
                </div>
                
                <div class="setting-item">
                    <div class="setting-info">
                        <h4>Maximum Meetings in Menu</h4>
                        <p>Limit the number of meetings shown in the tray menu</p>
                    </div>
                    <div class="setting-control">
                        <div class="form-group" style="margin: 0; width: 100px;">
                            <select id="maxMeetings">
                                <option value="3" {{if eq .Config.MaxMeetings 3}}selected{{end}}>3</option>
                                <option value="5" {{if eq .Config.MaxMeetings 5}}selected{{end}}>5</option>
                                <option value="10" {{if eq .Config.MaxMeetings 10}}selected{{end}}>10</option>
                                <option value="15" {{if eq .Config.MaxMeetings 15}}selected{{end}}>15</option>
                            </select>
                        </div>
                    </div>
                </div>
            </div>
            
            <div class="settings-section">
                <h3><span class="icon">🚀</span> Startup Settings</h3>
                
                <div class="setting-item">
                    <div class="setting-info">
                        <h4>Start with System</h4>
                        <p>Automatically start MeetingBar when you log in</p>
                    </div>
                    <div class="setting-control">
                        <label class="toggle">
                            <input type="checkbox" id="startWithSystem" {{if .Config.StartWithSystem}}checked{{end}}>
                            <span class="slider"></span>
                        </label>
                    </div>
                </div>
                
                <div class="setting-item">
                    <div class="setting-info">
                        <h4>Auto-refresh on Startup</h4>
                        <p>Immediately check for meetings when starting the app</p>
                    </div>
                    <div class="setting-control">
                        <label class="toggle">
                            <input type="checkbox" id="autoRefreshStartup" {{if .Config.AutoRefreshStartup}}checked{{end}}>
                            <span class="slider"></span>
                        </label>
                    </div>
                </div>
            </div>
            
            <div class="settings-section">
                <h3><span class="icon">📄</span> Configuration File</h3>
                
                <div class="form-group">
                    <label>Current Configuration:</label>
                    <div class="config-viewer">{{.ConfigJSON}}</div>
                </div>
            </div>
            
            <div class="actions">
                <button class="btn btn-success" onclick="saveGeneralSettings()">💾 Save Settings</button>
                <button class="btn" onclick="resetToDefaults()">🔄 Reset to Defaults</button>
                <button class="btn btn-danger" onclick="clearAllData()">🗑️ Clear All Data</button>
            </div>
        </div>
    </div>
    
    <script>
        async function saveGeneralSettings() {
            const settings = {
                refreshInterval: parseInt(document.getElementById('refreshInterval').value),
                showDuration: document.getElementById('showDuration').checked,
                maxMeetings: parseInt(document.getElementById('maxMeetings').value),
                startWithSystem: document.getElementById('startWithSystem').checked,
                autoRefreshStartup: document.getElementById('autoRefreshStartup').checked
            };
            
            try {
                const response = await fetch('/api/general', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ action: 'save', settings: settings })
                });
                
                const result = await response.json();
                
                if (result.success) {
                    alert('✅ General settings saved successfully!');
                    location.reload(); // Refresh to show updated config
                } else {
                    alert('❌ Error: ' + result.message);
                }
            } catch (error) {
                alert('❌ Error saving settings: ' + error.message);
            }
        }
        
        async function resetToDefaults() {
            if (!confirm('Are you sure you want to reset all settings to defaults? This will not affect your accounts or OAuth2 credentials.')) {
                return;
            }
            
            try {
                const response = await fetch('/api/general', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ action: 'reset' })
                });
                
                const result = await response.json();
                
                if (result.success) {
                    alert('✅ Settings reset to defaults!');
                    location.reload();
                } else {
                    alert('❌ Error: ' + result.message);
                }
            } catch (error) {
                alert('❌ Error resetting settings: ' + error.message);
            }
        }
        
        async function clearAllData() {
            if (!confirm('Are you sure you want to clear ALL data? This will remove accounts, OAuth2 credentials, and all settings. This action cannot be undone!')) {
                return;
            }
            
            if (!confirm('This will completely reset MeetingBar. Are you absolutely sure?')) {
                return;
            }
            
            try {
                const response = await fetch('/api/general', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ action: 'clear' })
                });
                
                const result = await response.json();
                
                if (result.success) {
                    alert('✅ All data cleared!');
                    location.reload();
                } else {
                    alert('❌ Error: ' + result.message);
                }
            } catch (error) {
                alert('❌ Error clearing data: ' + error.message);
            }
        }
    </script>
</body>
</html>`

	data := struct {
		Config     *config.Config
		ConfigJSON string
	}{
		Config:     wsm.config,
		ConfigJSON: wsm.getConfigJSON(),
	}

	t, err := template.New("general").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	t.Execute(w, data)
}

// Placeholder API handlers
func (wsm *WebSettingsManager) handleAccountsAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Not implemented yet"})
}

func (wsm *WebSettingsManager) handleCalendarsAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		Action            string   `json:"action"`
		SelectedCalendars []string `json:"selectedCalendars"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Invalid JSON"})
		return
	}

	switch data.Action {
	case "save":
		// Update enabled calendars
		wsm.config.EnabledCalendars = data.SelectedCalendars
		
		// Save configuration
		if err := wsm.config.Save(); err != nil {
			json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Failed to save configuration"})
			return
		}
		
		json.NewEncoder(w).Encode(APIResponse{Success: true, Message: "Calendar selection saved successfully"})
		
	default:
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Invalid action"})
	}
}

func (wsm *WebSettingsManager) handleNotificationsAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		Action   string `json:"action"`
		Settings struct {
			EnableNotifications      bool `json:"enableNotifications"`
			NotificationTime         int  `json:"notificationTime"`
			ShowMeetingLinks         bool `json:"showMeetingLinks"`
			PersistentNotifications  bool `json:"persistentNotifications"`
			NotificationSound        bool `json:"notificationSound"`
		} `json:"settings"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Invalid JSON"})
		return
	}

	switch data.Action {
	case "save":
		// Update notification settings
		wsm.config.EnableNotifications = data.Settings.EnableNotifications
		wsm.config.NotificationTime = data.Settings.NotificationTime
		wsm.config.ShowMeetingLinks = data.Settings.ShowMeetingLinks
		wsm.config.PersistentNotifications = data.Settings.PersistentNotifications
		wsm.config.NotificationSound = data.Settings.NotificationSound
		
		// Save configuration
		if err := wsm.config.Save(); err != nil {
			json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Failed to save configuration"})
			return
		}
		
		json.NewEncoder(w).Encode(APIResponse{Success: true, Message: "Notification settings saved successfully"})
		
	case "test":
		// Send test notification
		if wsm.notificationMgr != nil {
			testMeeting := calendar.Meeting{
				Title:     "Test Meeting",
				StartTime: time.Now().Add(5 * time.Minute),
				EndTime:   time.Now().Add(65 * time.Minute),
			}
			
			err := wsm.notificationMgr.ShowNotification(&testMeeting)
			if err != nil {
				json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Failed to send test notification: " + err.Error()})
				return
			}
		}
		
		json.NewEncoder(w).Encode(APIResponse{Success: true, Message: "Test notification sent"})
		
	default:
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Invalid action"})
	}
}

func (wsm *WebSettingsManager) handleGeneralAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		Action   string `json:"action"`
		Settings struct {
			RefreshInterval     int  `json:"refreshInterval"`
			ShowDuration        bool `json:"showDuration"`
			MaxMeetings         int  `json:"maxMeetings"`
			StartWithSystem     bool `json:"startWithSystem"`
			AutoRefreshStartup  bool `json:"autoRefreshStartup"`
		} `json:"settings"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Invalid JSON"})
		return
	}

	switch data.Action {
	case "save":
		// Update general settings
		wsm.config.RefreshInterval = data.Settings.RefreshInterval
		wsm.config.ShowDuration = data.Settings.ShowDuration
		wsm.config.MaxMeetings = data.Settings.MaxMeetings
		wsm.config.StartWithSystem = data.Settings.StartWithSystem
		wsm.config.AutoRefreshStartup = data.Settings.AutoRefreshStartup
		
		// Save configuration
		if err := wsm.config.Save(); err != nil {
			json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Failed to save configuration"})
			return
		}
		
		json.NewEncoder(w).Encode(APIResponse{Success: true, Message: "General settings saved successfully"})
		
	case "reset":
		// Reset to defaults (preserve OAuth2 and accounts)
		oauth2 := wsm.config.OAuth2
		accounts := wsm.config.Accounts
		
		// Reset config to defaults
		wsm.config = config.NewConfig()
		
		// Restore OAuth2 and accounts
		wsm.config.OAuth2 = oauth2
		wsm.config.Accounts = accounts
		
		// Save
		if err := wsm.config.Save(); err != nil {
			json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Failed to save configuration"})
			return
		}
		
		json.NewEncoder(w).Encode(APIResponse{Success: true, Message: "Settings reset to defaults"})
		
	case "clear":
		// Clear all data
		wsm.config = config.NewConfig()
		
		// Save empty config
		if err := wsm.config.Save(); err != nil {
			json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Failed to save configuration"})
			return
		}
		
		json.NewEncoder(w).Encode(APIResponse{Success: true, Message: "All data cleared"})
		
	default:
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Invalid action"})
	}
}

func (wsm *WebSettingsManager) handleAddAccountAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if OAuth2 credentials are configured
	if wsm.config.OAuth2.ClientID == "" || wsm.config.OAuth2.ClientSecret == "" {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "OAuth2 credentials not configured"})
		return
	}

	// Start the full OAuth2 flow (this includes starting the callback server)
	go func() {
		account, err := calendar.StartOAuth2Flow(wsm.ctx, wsm.config)
		if err != nil {
			log.Printf("OAuth2 flow failed: %v", err)
			return
		}
		
		// Add account to config
		wsm.config.Accounts = append(wsm.config.Accounts, *account)
		if err := wsm.config.Save(); err != nil {
			log.Printf("Failed to save config after adding account: %v", err)
			return
		}
		
		log.Printf("Successfully added account: %s", account.Email)
	}()

	// Generate OAuth URL for immediate redirect
	authURL, err := wsm.calendarService.GetAuthURL()
	if err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Failed to generate auth URL: " + err.Error()})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Message: "Authentication flow started",
		Data: map[string]string{"authUrl": authURL},
	})
}

func (wsm *WebSettingsManager) handleRemoveAccountAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		AccountID string `json:"accountId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Invalid JSON"})
		return
	}

	if data.AccountID == "" {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Account ID is required"})
		return
	}

	// Find and remove account
	found := false
	for i, account := range wsm.config.Accounts {
		if account.ID == data.AccountID {
			// Remove from slice
			wsm.config.Accounts = append(wsm.config.Accounts[:i], wsm.config.Accounts[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Account not found"})
		return
	}

	// Save configuration
	if err := wsm.config.Save(); err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "Failed to save configuration"})
		return
	}

	// Remove from keyring (optional, ignore errors)
	wsm.calendarService.RemoveAccount(data.AccountID)

	json.NewEncoder(w).Encode(APIResponse{Success: true, Message: "Account removed successfully"})
}

// Helper methods
func (wsm *WebSettingsManager) getNotificationStatus() string {
	if wsm.config.EnableNotifications {
		return fmt.Sprintf("✅ %dm before", wsm.config.NotificationTime)
	}
	return "❌ Disabled"
}

func (wsm *WebSettingsManager) getClientIDPreview() string {
	if wsm.config.OAuth2.ClientID == "" {
		return ""
	}
	if len(wsm.config.OAuth2.ClientID) > 16 {
		return wsm.config.OAuth2.ClientID[:8] + "..." + wsm.config.OAuth2.ClientID[len(wsm.config.OAuth2.ClientID)-8:]
	}
	return wsm.config.OAuth2.ClientID
}

func (wsm *WebSettingsManager) getAccountsInfo() []AccountInfo {
	var accounts []AccountInfo
	for _, account := range wsm.config.Accounts {
		// Get first letter for avatar
		avatar := "?"
		if len(account.Email) > 0 {
			avatar = string(account.Email[0])
		}
		
		accounts = append(accounts, AccountInfo{
			ID:      account.ID,
			Email:   account.Email,
			Avatar:  avatar,
			AddedAt: account.AddedAt.Format("Jan 2, 2006"),
		})
	}
	return accounts
}

func (wsm *WebSettingsManager) getAccountCalendarsInfo() []AccountCalendarsInfo {
	var accountCalendars []AccountCalendarsInfo
	
	for _, account := range wsm.config.Accounts {
		// Get first letter for avatar
		avatar := "?"
		if len(account.Email) > 0 {
			avatar = string(account.Email[0])
		}
		
		// Get calendars for this account
		calendars, err := wsm.calendarService.GetCalendars(account.ID)
		if err != nil {
			log.Printf("Failed to get calendars for account %s: %v", account.Email, err)
			continue
		}
		
		var calendarInfos []CalendarInfo
		for _, cal := range calendars {
			// Check if calendar is selected
			selected := false
			for _, enabledID := range wsm.config.EnabledCalendars {
				if enabledID == cal.ID {
					selected = true
					break
				}
			}
			
			// Default color if not provided
			color := cal.BackgroundColor
			if color == "" {
				color = "#3b82f6"
			}
			
			description := cal.Description
			if description == "" {
				description = "Google Calendar"
			}
			
			calendarInfos = append(calendarInfos, CalendarInfo{
				ID:          cal.ID,
				Title:       cal.Summary,
				Description: description,
				Color:       color,
				Selected:    selected,
			})
		}
		
		accountCalendars = append(accountCalendars, AccountCalendarsInfo{
			Email:         account.Email,
			Avatar:        avatar,
			CalendarCount: len(calendarInfos),
			Calendars:     calendarInfos,
		})
	}
	
	return accountCalendars
}

func (wsm *WebSettingsManager) getNotificationPreview() string {
	if !wsm.config.EnableNotifications {
		return "Notifications: Disabled"
	}
	
	preview := fmt.Sprintf("Notifications: Enabled, %d minutes before meetings", wsm.config.NotificationTime)
	if wsm.config.ShowMeetingLinks {
		preview += ", with meeting links"
	}
	if wsm.config.PersistentNotifications {
		preview += ", persistent"
	}
	if wsm.config.NotificationSound {
		preview += ", with sound"
	}
	
	return preview
}

func (wsm *WebSettingsManager) getConfigJSON() string {
	configBytes, err := json.MarshalIndent(wsm.config, "", "  ")
	if err != nil {
		return "Error marshaling config: " + err.Error()
	}
	return string(configBytes)
}

func (wsm *WebSettingsManager) handleOAuthSuccess(w http.ResponseWriter, r *http.Request) {
	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Account Added - MeetingBar</title>
    <style>
        body { font-family: system-ui; text-align: center; padding: 50px; background: #f0f9ff; }
        .success { background: #10b981; color: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; }
        .btn { background: #3b82f6; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; display: inline-block; }
    </style>
</head>
<body>
    <div class="success">
        <h2>✅ Account Added Successfully!</h2>
        <p>Your Google account has been added to MeetingBar.</p>
    </div>
    <a href="/accounts" class="btn">Return to Accounts</a>
    <script>
        // Auto-close after 5 seconds
        setTimeout(() => {
            window.location.href = '/accounts';
        }, 5000);
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(tmpl))
}