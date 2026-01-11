package cmd

import (
	"fmt"

	"github.com/gwyn/gh-subissue/internal/api"
	"github.com/gwyn/gh-subissue/internal/debug"
)

// SelectParentIssue prompts user to select an issue from a list.
// Returns the selected issue number.
func SelectParentIssue(p Prompter, issues []api.Issue) (int, error) {
	debug.Log("SelectParentIssue", "issue_count", len(issues))

	if len(issues) == 0 {
		err := fmt.Errorf("no open issues found in repository\n\nTo create a parent issue first:\n  gh issue create\n\nOr use --parent with an existing issue number")
		debug.Error("SelectParentIssue", err)
		return 0, err
	}

	options := make([]string, len(issues))
	for i, issue := range issues {
		// Format: "#123 Issue title" (truncate title if needed)
		title := issue.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		options[i] = fmt.Sprintf("#%d %s", issue.Number, title)
	}

	debug.Log("SelectParentIssue", "action", "prompting_user", "options_count", len(options))
	idx, err := p.Select("Select parent issue", "", options)
	if err != nil {
		debug.Error("SelectParentIssue", err, "stage", "prompt_select")
		return 0, err
	}

	selectedNumber := issues[idx].Number
	debug.Log("SelectParentIssue", "selected_index", idx, "selected_number", selectedNumber)
	return selectedNumber, nil
}

// SelectProject prompts user to select a project from a list.
// Returns the selected project.
func SelectProject(p Prompter, projects []api.Project) (*api.Project, error) {
	debug.Log("SelectProject", "project_count", len(projects))

	if len(projects) == 0 {
		err := fmt.Errorf("no projects found for this repository")
		debug.Error("SelectProject", err)
		return nil, err
	}

	options := make([]string, len(projects))
	for i, project := range projects {
		// Format: "Project Title (#N)"
		options[i] = fmt.Sprintf("%s (#%d)", project.Title, project.Number)
	}

	debug.Log("SelectProject", "action", "prompting_user", "options_count", len(options))
	idx, err := p.Select("Select project", "", options)
	if err != nil {
		debug.Error("SelectProject", err, "stage", "prompt_select")
		return nil, err
	}

	selected := &projects[idx]
	debug.Log("SelectProject", "selected_index", idx, "selected_project", selected.Title)
	return selected, nil
}
