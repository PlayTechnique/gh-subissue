package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Client handles GitHub API requests.
type Client struct {
	HTTPClient *http.Client
	BaseURL    string
}

// CreateIssueOptions contains parameters for creating an issue.
type CreateIssueOptions struct {
	Owner     string
	Repo      string
	Title     string
	Body      string
	Labels    []string
	Assignees []string
	Milestone int
}

// IssueResult contains the response from creating an issue.
type IssueResult struct {
	ID     int64
	Number int
	URL    string
}

// LinkSubIssueOptions contains parameters for linking a sub-issue.
type LinkSubIssueOptions struct {
	Owner       string
	Repo        string
	ParentIssue int
	SubIssueID  int64
}

// Issue represents a GitHub issue.
type Issue struct {
	ID     int64  `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"html_url"`
}

// CreateIssue creates a new issue in the specified repository.
func (c *Client) CreateIssue(opts CreateIssueOptions) (*IssueResult, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues", c.BaseURL, opts.Owner, opts.Repo)

	payload := map[string]interface{}{
		"title": opts.Title,
	}
	if opts.Body != "" {
		payload["body"] = opts.Body
	}
	if len(opts.Labels) > 0 {
		payload["labels"] = opts.Labels
	}
	if len(opts.Assignees) > 0 {
		payload["assignees"] = opts.Assignees
	}
	if opts.Milestone > 0 {
		payload["milestone"] = opts.Milestone
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("failed to create issue: %s (status %d)", errResp["message"], resp.StatusCode)
	}

	var result struct {
		ID     int64  `json:"id"`
		Number int    `json:"number"`
		URL    string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &IssueResult{
		ID:     result.ID,
		Number: result.Number,
		URL:    result.URL,
	}, nil
}

// LinkSubIssue links a sub-issue to a parent issue.
func (c *Client) LinkSubIssue(opts LinkSubIssueOptions) error {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/sub_issues", c.BaseURL, opts.Owner, opts.Repo, opts.ParentIssue)

	payload := map[string]interface{}{
		"sub_issue_id": opts.SubIssueID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed to link sub-issue: %s (status %d)", errResp["message"], resp.StatusCode)
	}

	return nil
}

// GetIssue retrieves an issue by number.
func (c *Client) GetIssue(owner, repo string, number int) (*Issue, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d", c.BaseURL, owner, repo, number)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("failed to get issue: %s (status %d)", errResp["message"], resp.StatusCode)
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &issue, nil
}
