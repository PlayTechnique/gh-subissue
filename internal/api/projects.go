package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gwyn/gh-subissue/internal/debug"
)

// Project represents a GitHub project (v2).
type Project struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Number int    `json:"number"`
}

// graphqlRequest executes a GraphQL query against the GitHub API.
func (c *Client) graphqlRequest(query string, variables map[string]interface{}) (map[string]interface{}, error) {
	// Determine GraphQL endpoint
	graphqlURL := "https://api.github.com/graphql"
	if c.BaseURL != "https://api.github.com" && c.BaseURL != "" {
		// For GitHub Enterprise or test servers, use the base URL
		graphqlURL = c.BaseURL + "/graphql"
	}

	payload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GraphQL request: %w", err)
	}

	req, err := http.NewRequest("POST", graphqlURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create GraphQL request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send GraphQL request: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode GraphQL response: %w", err)
	}

	// Check for GraphQL errors
	if errors, ok := result["errors"].([]interface{}); ok && len(errors) > 0 {
		if firstErr, ok := errors[0].(map[string]interface{}); ok {
			if msg, ok := firstErr["message"].(string); ok {
				return nil, fmt.Errorf("GraphQL error: %s", msg)
			}
		}
		return nil, fmt.Errorf("GraphQL error: %v", errors)
	}

	return result, nil
}

// ListProjects returns projects associated with a repository.
func (c *Client) ListProjects(owner, repo string) ([]Project, error) {
	debug.Log("ListProjects", "owner", owner, "repo", repo)

	query := `
		query($owner: String!, $repo: String!) {
			repository(owner: $owner, name: $repo) {
				projectsV2(first: 20) {
					nodes {
						id
						title
						number
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"owner": owner,
		"repo":  repo,
	}

	result, err := c.graphqlRequest(query, variables)
	if err != nil {
		debug.Error("ListProjects", err, "stage", "graphql_request")
		return nil, err
	}

	// Parse the response
	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	repository, ok := data["repository"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("repository not found in response")
	}

	projectsV2, ok := repository["projectsV2"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("projectsV2 not found in response")
	}

	nodes, ok := projectsV2["nodes"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("nodes not found in response")
	}

	projects := make([]Project, 0, len(nodes))
	for _, node := range nodes {
		if n, ok := node.(map[string]interface{}); ok {
			p := Project{}
			if id, ok := n["id"].(string); ok {
				p.ID = id
			}
			if title, ok := n["title"].(string); ok {
				p.Title = title
			}
			if number, ok := n["number"].(float64); ok {
				p.Number = int(number)
			}
			projects = append(projects, p)
		}
	}

	debug.Log("ListProjects", "result_count", len(projects))
	return projects, nil
}

// GetIssueNodeID retrieves the GraphQL node ID for an issue.
func (c *Client) GetIssueNodeID(owner, repo string, number int) (string, error) {
	debug.Log("GetIssueNodeID", "owner", owner, "repo", repo, "number", number)

	query := `
		query($owner: String!, $repo: String!, $number: Int!) {
			repository(owner: $owner, name: $repo) {
				issue(number: $number) {
					id
				}
			}
		}
	`

	variables := map[string]interface{}{
		"owner":  owner,
		"repo":   repo,
		"number": number,
	}

	result, err := c.graphqlRequest(query, variables)
	if err != nil {
		debug.Error("GetIssueNodeID", err, "stage", "graphql_request")
		return "", err
	}

	// Parse the response
	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected response format")
	}

	repository, ok := data["repository"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("repository not found in response")
	}

	issue, ok := repository["issue"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("issue not found in response")
	}

	nodeID, ok := issue["id"].(string)
	if !ok {
		return "", fmt.Errorf("issue id not found in response")
	}

	debug.Log("GetIssueNodeID", "result_node_id", nodeID)
	return nodeID, nil
}

// AddIssueToProject adds an issue to a project using GraphQL mutation.
func (c *Client) AddIssueToProject(projectID, issueNodeID string) error {
	debug.Log("AddIssueToProject", "project_id", projectID, "issue_node_id", issueNodeID)

	query := `
		mutation($projectId: ID!, $contentId: ID!) {
			addProjectV2ItemById(input: {projectId: $projectId, contentId: $contentId}) {
				item {
					id
				}
			}
		}
	`

	variables := map[string]interface{}{
		"projectId": projectID,
		"contentId": issueNodeID,
	}

	_, err := c.graphqlRequest(query, variables)
	if err != nil {
		debug.Error("AddIssueToProject", err, "stage", "graphql_request")
		return err
	}

	debug.Log("AddIssueToProject", "result", "success")
	return nil
}
