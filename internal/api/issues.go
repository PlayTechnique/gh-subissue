package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gwyn/gh-subissue/internal/debug"
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
	debug.Log("CreateIssue", "owner", opts.Owner, "repo", opts.Repo, "title", opts.Title)

	url := fmt.Sprintf("%s/repos/%s/%s/issues", c.BaseURL, opts.Owner, opts.Repo)
	debug.Log("CreateIssue", "url", url)

	payload := map[string]interface{}{
		"title": opts.Title,
	}
	if opts.Body != "" {
		payload["body"] = opts.Body
	}
	if len(opts.Labels) > 0 {
		payload["labels"] = opts.Labels
		debug.Log("CreateIssue", "labels", opts.Labels)
	}
	if len(opts.Assignees) > 0 {
		payload["assignees"] = opts.Assignees
		debug.Log("CreateIssue", "assignees", opts.Assignees)
	}
	if opts.Milestone > 0 {
		payload["milestone"] = opts.Milestone
		debug.Log("CreateIssue", "milestone", opts.Milestone)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		debug.Error("CreateIssue", err, "stage", "marshal")
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		debug.Error("CreateIssue", err, "stage", "new_request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	debug.Log("CreateIssue", "action", "sending_request")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		debug.Error("CreateIssue", err, "stage", "do_request")
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	debug.Log("CreateIssue", "status_code", resp.StatusCode)
	if resp.StatusCode != http.StatusCreated {
		var errResp struct {
			Message string `json:"message"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		apiErr := newAPIError(resp.StatusCode, errResp.Message, "create issue")
		debug.Error("CreateIssue", apiErr, "status", resp.StatusCode)
		return nil, apiErr
	}

	var result struct {
		ID     int64  `json:"id"`
		Number int    `json:"number"`
		URL    string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		debug.Error("CreateIssue", err, "stage", "decode_response")
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	debug.Log("CreateIssue", "created_id", result.ID, "created_number", result.Number, "url", result.URL)
	return &IssueResult{
		ID:     result.ID,
		Number: result.Number,
		URL:    result.URL,
	}, nil
}

// LinkSubIssue links a sub-issue to a parent issue.
func (c *Client) LinkSubIssue(opts LinkSubIssueOptions) error {
	debug.Log("LinkSubIssue", "owner", opts.Owner, "repo", opts.Repo, "parent_issue", opts.ParentIssue, "sub_issue_id", opts.SubIssueID)

	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/sub_issues", c.BaseURL, opts.Owner, opts.Repo, opts.ParentIssue)
	debug.Log("LinkSubIssue", "url", url)

	payload := map[string]interface{}{
		"sub_issue_id": opts.SubIssueID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		debug.Error("LinkSubIssue", err, "stage", "marshal")
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		debug.Error("LinkSubIssue", err, "stage", "new_request")
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	debug.Log("LinkSubIssue", "action", "sending_request")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		debug.Error("LinkSubIssue", err, "stage", "do_request")
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	debug.Log("LinkSubIssue", "status_code", resp.StatusCode)
	if resp.StatusCode != http.StatusCreated {
		var errResp struct {
			Message string `json:"message"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		apiErr := newAPIError(resp.StatusCode, errResp.Message, "link sub-issue")
		debug.Error("LinkSubIssue", apiErr, "status", resp.StatusCode)
		return apiErr
	}

	debug.Log("LinkSubIssue", "result", "success")
	return nil
}

// ListIssuesOptions contains parameters for listing issues.
type ListIssuesOptions struct {
	Owner   string
	Repo    string
	State   string // "open", "closed", "all"
	PerPage int
}

// ListIssues lists issues in a repository.
func (c *Client) ListIssues(opts ListIssuesOptions) ([]Issue, error) {
	debug.Log("ListIssues", "owner", opts.Owner, "repo", opts.Repo, "state", opts.State, "per_page", opts.PerPage)

	url := fmt.Sprintf("%s/repos/%s/%s/issues?state=%s&per_page=%d",
		c.BaseURL, opts.Owner, opts.Repo, opts.State, opts.PerPage)
	debug.Log("ListIssues", "url", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		debug.Error("ListIssues", err, "stage", "new_request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	debug.Log("ListIssues", "action", "sending_request")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		debug.Error("ListIssues", err, "stage", "do_request")
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	debug.Log("ListIssues", "status_code", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Message string `json:"message"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		apiErr := newAPIError(resp.StatusCode, errResp.Message, "list issues")
		debug.Error("ListIssues", apiErr, "status", resp.StatusCode)
		return nil, apiErr
	}

	var issues []Issue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		debug.Error("ListIssues", err, "stage", "decode_response")
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	debug.Log("ListIssues", "result_count", len(issues))
	return issues, nil
}

// GetIssue retrieves an issue by number.
func (c *Client) GetIssue(owner, repo string, number int) (*Issue, error) {
	debug.Log("GetIssue", "owner", owner, "repo", repo, "number", number)

	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d", c.BaseURL, owner, repo, number)
	debug.Log("GetIssue", "url", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		debug.Error("GetIssue", err, "stage", "new_request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	debug.Log("GetIssue", "action", "sending_request")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		debug.Error("GetIssue", err, "stage", "do_request")
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	debug.Log("GetIssue", "status_code", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Message string `json:"message"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		apiErr := newAPIError(resp.StatusCode, errResp.Message, fmt.Sprintf("get issue #%d", number))
		debug.Error("GetIssue", apiErr, "status", resp.StatusCode)
		return nil, apiErr
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		debug.Error("GetIssue", err, "stage", "decode_response")
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	debug.Log("GetIssue", "result_id", issue.ID, "result_number", issue.Number, "title", issue.Title)
	return &issue, nil
}
