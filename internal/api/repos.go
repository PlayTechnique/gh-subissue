package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gwyn/gh-subissue/internal/debug"
)

// Repository represents a GitHub repository.
type Repository struct {
	Name      string `json:"name"`
	FullName  string `json:"full_name"`
	HasIssues bool   `json:"has_issues"`
	Archived  bool   `json:"archived"`
	Private   bool   `json:"private"`
}

// ListRepositoriesOptions contains parameters for listing repositories.
type ListRepositoriesOptions struct {
	Owner   string
	PerPage int
	Page    int
}

// User represents a GitHub user.
type User struct {
	Login string `json:"login"`
}

// ListRepositories lists repositories for an owner (user or organization).
// It tries the org endpoint first, then falls back to the user endpoint.
func (c *Client) ListRepositories(opts ListRepositoriesOptions) ([]Repository, error) {
	debug.Log("ListRepositories", "owner", opts.Owner, "per_page", opts.PerPage, "page", opts.Page)

	// Try org endpoint first
	repos, err := c.listOrgRepositories(opts)
	if err == nil {
		return repos, nil
	}

	// Check if it was a 404 (org not found), fall back to user endpoint
	if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == http.StatusNotFound {
		debug.Log("ListRepositories", "action", "fallback_to_user", "reason", "org_not_found")
		return c.listUserRepositories(opts)
	}

	return nil, err
}

func (c *Client) listOrgRepositories(opts ListRepositoriesOptions) ([]Repository, error) {
	url := fmt.Sprintf("%s/orgs/%s/repos?per_page=%d", c.BaseURL, opts.Owner, opts.PerPage)
	if opts.Page > 0 {
		url = fmt.Sprintf("%s&page=%d", url, opts.Page)
	}
	debug.Log("listOrgRepositories", "url", url)

	return c.fetchRepositories(url, "list org repositories")
}

func (c *Client) listUserRepositories(opts ListRepositoriesOptions) ([]Repository, error) {
	url := fmt.Sprintf("%s/users/%s/repos?per_page=%d", c.BaseURL, opts.Owner, opts.PerPage)
	if opts.Page > 0 {
		url = fmt.Sprintf("%s&page=%d", url, opts.Page)
	}
	debug.Log("listUserRepositories", "url", url)

	return c.fetchRepositories(url, "list user repositories")
}

func (c *Client) fetchRepositories(url, operation string) ([]Repository, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		debug.Error("fetchRepositories", err, "stage", "new_request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	debug.Log("fetchRepositories", "action", "sending_request")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		debug.Error("fetchRepositories", err, "stage", "do_request")
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	debug.Log("fetchRepositories", "status_code", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Message string `json:"message"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		apiErr := newAPIError(resp.StatusCode, errResp.Message, operation)
		debug.Error("fetchRepositories", apiErr, "status", resp.StatusCode)
		return nil, apiErr
	}

	var repos []Repository
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		debug.Error("fetchRepositories", err, "stage", "decode_response")
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	debug.Log("fetchRepositories", "result_count", len(repos))
	return repos, nil
}

// GetAuthenticatedUser returns the currently authenticated user.
func (c *Client) GetAuthenticatedUser() (*User, error) {
	debug.Log("GetAuthenticatedUser", "action", "start")

	url := fmt.Sprintf("%s/user", c.BaseURL)
	debug.Log("GetAuthenticatedUser", "url", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		debug.Error("GetAuthenticatedUser", err, "stage", "new_request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	debug.Log("GetAuthenticatedUser", "action", "sending_request")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		debug.Error("GetAuthenticatedUser", err, "stage", "do_request")
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	debug.Log("GetAuthenticatedUser", "status_code", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Message string `json:"message"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		apiErr := newAPIError(resp.StatusCode, errResp.Message, "get authenticated user")
		debug.Error("GetAuthenticatedUser", apiErr, "status", resp.StatusCode)
		return nil, apiErr
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		debug.Error("GetAuthenticatedUser", err, "stage", "decode_response")
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	debug.Log("GetAuthenticatedUser", "login", user.Login)
	return &user, nil
}
