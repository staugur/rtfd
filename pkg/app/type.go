package app

// GitHub App Post Webhook data
type Webhook struct {
	// Install / Uninstall (nonexistent app)
	// Suspend / Unsuspend (existing app)
	Action       string       `json:"action"`
	Installation Installation `json:"installation"`
	Repositories []Repository `json:"repositories"`
	Sender       Sender       `json:"sender"`
	// Add / Remove repo in an existing app
	Repositories_removed []Repository `json:"repositories_removed"`
	Repositories_added   []Repository `json:"repositories_added"`
}

// Data structure after the user installs the github app
type Installation struct {
	ID             uint64 `json:"id"`
	AppID          uint64 `json:"app_id"`
	AppName        string `json:"app_slug"`
	AccessTokenURL string `json:"access_tokens_url"`
}

// Repo name & id
type Repository struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	ID       uint64 `json:"id"`
}

// GitHub User info
type Sender struct {
	Type  string `json:"type"`
	Login string `json:"login"`
	ID    uint64 `json:"id"`
}

type AccessToken struct {
	Token string `json:"token"`
}

type UserWebhook struct {
	Type   string            `json:"type"`
	ID     uint64            `json:"id"`
	Name   string            `json:"name"`
	Active bool              `json:"active"`
	Events []string          `json:"events"`
	Config UserWebhookConfig `json:"config"`
}

type UserWebhookConfig struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Secret      string `json:"secret"`
}
