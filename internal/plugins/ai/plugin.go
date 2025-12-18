package ai

import (
	"context"
	"strings"
	"time"

	"dev-cli/internal/llm"
	"dev-cli/internal/pipeline"
)

type Plugin struct {
	bus      *pipeline.EventBus
	state    *pipeline.StateStore
	client   *llm.HybridClient
	patterns map[string]string
}

func New(client *llm.HybridClient) *Plugin {
	return &Plugin{
		client: client,
		patterns: map[string]string{
			"command not found":         "Check if the command is installed or if it's an alias",
			"permission denied":         "Try with sudo or check file permissions",
			"no such file or directory": "Check the path exists",
			"cannot find module":        "Run: npm install",
			"ModuleNotFoundError":       "Run: pip install <module>",
			"package.json":              "Run: npm init -y",
			"EACCES":                    "Try with sudo or fix permissions",
			"Connection refused":        "Check if the service is running",
			"docker: Error response":    "Check Docker daemon is running",
		},
	}
}

func (p *Plugin) Name() string {
	return "ai"
}

func (p *Plugin) Init(bus *pipeline.EventBus, state *pipeline.StateStore) error {
	p.bus = bus
	p.state = state

	bus.Subscribe(pipeline.EventCommandError, p.handleCommandError)

	return nil
}

func (p *Plugin) Start(ctx context.Context) error {
	return nil
}

func (p *Plugin) Stop() error {
	return nil
}

func (p *Plugin) handleCommandError(event pipeline.Event) {
	block, ok := event.Data.(pipeline.Block)
	if !ok {
		return
	}

	suggestion := p.matchPattern(block.Output)

	if suggestion != "" {
		p.state.AddSuggestion(pipeline.Suggestion{
			ForBlockID:  block.ID,
			Type:        "fix",
			Title:       "Quick Fix",
			Explanation: suggestion,
			Confidence:  0.8,
		})

		p.bus.Publish(pipeline.Event{
			Type:      pipeline.EventAISuggestion,
			Timestamp: time.Now(),
			Source:    p.Name(),
			BlockID:   block.ID,
			Data: map[string]string{
				"suggestion": suggestion,
			},
		})
	}

}

func (p *Plugin) matchPattern(output string) string {
	lowerOutput := strings.ToLower(output)

	for pattern, fix := range p.patterns {
		if strings.Contains(lowerOutput, strings.ToLower(pattern)) {
			return fix
		}
	}
	return ""
}

func (p *Plugin) AnalyzeError(block pipeline.Block) (*pipeline.Suggestion, error) {
	if p.client == nil {
		return nil, nil
	}

	result, err := p.client.Research(
		"Fix this command error: " + block.Command + "\n\nError: " + block.Output,
	)
	if err != nil {
		return nil, err
	}

	var fix string
	if len(result.Solutions) > 0 && len(result.Solutions[0].Steps) > 0 {
		fix = result.Solutions[0].Steps[0].Content
	}

	suggestion := &pipeline.Suggestion{
		ForBlockID:  block.ID,
		Type:        "fix",
		Title:       "AI Suggestion",
		Command:     fix,
		Explanation: result.Query,
		Confidence:  0.7,
	}

	p.state.AddSuggestion(*suggestion)
	return suggestion, nil
}

func (p *Plugin) AnswerQuery(query string, blockID string) (string, error) {
	if p.client == nil {
		return "AI client not available", nil
	}

	context := p.state.GetContext()

	enrichedQuery := query
	if context["git_branch"] != "" {
		enrichedQuery += " (in git repo: " + context["git_branch"].(string) + ")"
	}

	result, err := p.client.Research(enrichedQuery)
	if err != nil {
		return "", err
	}

	var response strings.Builder
	for _, sol := range result.Solutions {
		response.WriteString("### " + sol.Title + "\n")
		response.WriteString(sol.Description + "\n\n")
		for _, step := range sol.Steps {
			if step.Type == "command" {
				response.WriteString("```\n" + step.Content + "\n```\n")
			} else {
				response.WriteString(step.Content + "\n")
			}
		}
		response.WriteString("\n")
	}

	p.state.UpdateBlock(blockID, func(b *pipeline.Block) {
		b.Output = response.String()
	})

	return response.String(), nil
}
