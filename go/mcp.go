package streamoid

import "context"

// RegisterMCPServerOptions carries the fields unified-backend's mcp_servers
// registry stores (upsert by owner+serverId).
type RegisterMCPServerOptions struct {
	ServerID      string
	Owner         string
	DisplayName   string
	URL           string
	Transport     string
	HealthURL     string
	AuthSecretRef string
	Scopes        []string
	ToolSummary   []map[string]string
	Version       string
	Enabled       bool
}

type registerMCPServerRequest struct {
	ServerID    string              `json:"serverId"`
	Owner       string              `json:"owner"`
	DisplayName string              `json:"displayName"`
	URL         string              `json:"url"`
	Transport   string              `json:"transport"`
	HealthURL   string              `json:"healthUrl,omitempty"`
	Auth        *mcpAuth            `json:"auth,omitempty"`
	Scopes      []string            `json:"scopes,omitempty"`
	ToolSummary []map[string]string `json:"toolSummary,omitempty"`
	Version     string              `json:"version,omitempty"`
	Enabled     bool                `json:"enabled"`
}

type mcpAuth struct {
	SecretRef string `json:"secretRef"`
}

// RegisterMCPServer POSTs this server's registration to the platform MCP
// registry (POST /api/v1/mcp/servers, upsert by owner+serverId). Best-
// effort: soft-skips if no platform API URL is configured; a registry that's
// briefly down must not stop the calling service from booting or serving
// traffic, so callers should log-and-continue on a non-nil error rather than
// fail startup.
func (c *Client) RegisterMCPServer(ctx context.Context, opts RegisterMCPServerOptions) error {
	if c == nil || !c.cfg.HasPlatform() {
		return nil
	}
	req := registerMCPServerRequest{
		ServerID:    opts.ServerID,
		Owner:       opts.Owner,
		DisplayName: opts.DisplayName,
		URL:         opts.URL,
		Transport:   opts.Transport,
		HealthURL:   opts.HealthURL,
		Scopes:      opts.Scopes,
		ToolSummary: opts.ToolSummary,
		Version:     opts.Version,
		Enabled:     opts.Enabled,
	}
	if opts.AuthSecretRef != "" {
		req.Auth = &mcpAuth{SecretRef: opts.AuthSecretRef}
	}
	url := c.cfg.MCPRegistryURL + "/api/v1/mcp/servers"
	return c.postJSON(ctx, url, req, nil)
}
