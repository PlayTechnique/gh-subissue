package cmd

// Prompter handles interactive user prompts.
type Prompter interface {
	Select(prompt string, defaultValue string, options []string) (int, error)
}
