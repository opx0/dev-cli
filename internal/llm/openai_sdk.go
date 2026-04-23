package llm

import (
	"context"
	"fmt"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/shared"
)

// chatCompletionViaSDK sends a ChatRequest through the openai-go SDK and maps
// the response back to our provider-agnostic ChatResponse. Shared by the
// Ollama provider (local OpenAI-compatible endpoint) and the generic OpenAI
// provider (cloud / OpenAI-compatible endpoints).
func chatCompletionViaSDK(ctx context.Context, client *openai.Client, req ChatRequest) (*ChatResponse, error) {
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(req.Messages))
	for _, m := range req.Messages {
		messages = append(messages, convertMessage(m))
	}

	params := openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(req.Model),
		Messages: messages,
	}

	if len(req.Tools) > 0 {
		tools := make([]openai.ChatCompletionToolParam, 0, len(req.Tools))
		for _, td := range req.Tools {
			tools = append(tools, openai.ChatCompletionToolParam{
				Function: shared.FunctionDefinitionParam{
					Name:        td.Name,
					Description: param.NewOpt(td.Description),
					Parameters:  shared.FunctionParameters(td.Parameters),
				},
			})
		}
		params.Tools = tools
	}

	if req.Temperature != nil {
		params.Temperature = param.NewOpt(*req.Temperature)
	}
	if req.MaxTokens != nil {
		params.MaxCompletionTokens = param.NewOpt(*req.MaxTokens)
	}

	completion, err := client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, err
	}

	if len(completion.Choices) == 0 {
		return nil, fmt.Errorf("provider returned no choices")
	}

	choice := completion.Choices[0]

	var toolCalls []ToolCall
	for _, tc := range choice.Message.ToolCalls {
		toolCalls = append(toolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	return &ChatResponse{
		Content:      choice.Message.Content,
		ToolCalls:    toolCalls,
		FinishReason: choice.FinishReason,
		Usage: Usage{
			PromptTokens:     completion.Usage.PromptTokens,
			CompletionTokens: completion.Usage.CompletionTokens,
			TotalTokens:      completion.Usage.TotalTokens,
		},
	}, nil
}
