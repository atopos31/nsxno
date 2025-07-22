package react

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/ssestream"
	"github.com/tidwall/gjson"
)

type ReactChunk struct {
	Type     string
	Content  string
	ToolCall openai.ChatCompletionChunkChoiceDeltaToolCall
	ToolRes  openai.ChatCompletionToolMessageParamContentUnion
	Step     int
}

type ToolHandle func(ctx context.Context, call openai.ChatCompletionChunkChoiceDeltaToolCallFunction) (*openai.ChatCompletionToolMessageParamContentUnion, error)

type Agent struct {
	ChatClient openai.Client
	MaxStep    int
}

func New(chatClient openai.Client, maxStep int) *Agent {
	return &Agent{
		ChatClient: chatClient,
		MaxStep:    maxStep,
	}
}

func (a *Agent) Run(ctx context.Context, params openai.ChatCompletionNewParams, toolHandler ToolHandle, reactChunk chan<- ReactChunk) ([]openai.ChatCompletionMessageParamUnion, error) {
	messages := params.Messages
	var index int
	for ; index < a.MaxStep; index++ {
		toolCalls := make(map[int64]*openai.ChatCompletionChunkChoiceDeltaToolCall)
		var acc openai.ChatCompletionAccumulator

		params.Messages = messages
		stream := a.ChatClient.Chat.Completions.NewStreaming(ctx, params)

		// 异步并发工具执行的 WaitGroup
		var wg sync.WaitGroup

		// 并发调用工具安全访问 toolMessages
		var toolmsgmu sync.Mutex
		toolMessages := make([]openai.ChatCompletionMessageParamUnion, 0)
		for chunk := range Chunks(stream) {
			acc.AddChunk(chunk)
			if len(chunk.Choices) < 1 {
				continue
			}
			content := chunk.Choices[0].Delta.Content
			if content != "" {
				reactChunk <- ReactChunk{
					Type:    "text",
					Content: content,
					Step:    index,
				}
			}
			for _, call := range chunk.Choices[0].Delta.ToolCalls {
				if _, ok := toolCalls[call.Index]; !ok {
					toolCalls[call.Index] = &call
					continue
				}
				toolCall := toolCalls[call.Index]
				toolCall.Function.Arguments += call.Function.Arguments
				//流式工具调用完成
				if gjson.Valid(toolCall.Function.Arguments) {
					wg.Add(1)
					go func() {
						defer wg.Done()
						reactChunk <- ReactChunk{
							Type:     "toolcall",
							ToolCall: *toolCall,
							Step:     index,
						}
						toolmsgmu.Lock()
						defer toolmsgmu.Unlock()
						res, err := toolHandler(ctx, toolCall.Function)
						if err != nil {
							toolMessages = append(toolMessages, openai.ToolMessage(err.Error(), toolCall.ID))
							return
						}
						var toolRes openai.ChatCompletionMessageParamUnion
						if len(res.OfArrayOfContentParts) > 0 {
							toolRes = openai.ToolMessage(res.OfArrayOfContentParts, toolCall.ID)
						} else {
							toolRes = openai.ToolMessage(res.OfString.Value, toolCall.ID)
						}
						toolMessages = append(toolMessages, toolRes)
						reactChunk <- ReactChunk{
							Type:    "toolres",
							ToolRes: *res,
							Step:    index,
						}
					}()
				}
			}
		}
		if err := stream.Err(); err != nil {
			return nil, err
		}
		messages = append(messages, acc.ChatCompletion.Choices[0].Message.ToParam())

		// 等待工具执行结束
		wg.Wait()
		// 不再有工具调用 结束React
		if len(toolCalls) == 0 {
			break
		}
		messages = append(messages, toolMessages...)
	}
	if index < a.MaxStep {
		return messages, nil
	}
	return nil, errors.New("limit max steps")
}

func McpToolHandler(session *mcp.ClientSession) ToolHandle {
	return func(ctx context.Context, call openai.ChatCompletionChunkChoiceDeltaToolCallFunction) (*openai.ChatCompletionToolMessageParamContentUnion, error) {
		var args json.RawMessage
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return nil, err
		}

		res, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      call.Name,
			Arguments: args,
		})
		if err != nil {
			return nil, err
		}
		content := ""
		for _, c := range res.Content {
			content += c.(*mcp.TextContent).Text
		}
		return &openai.ChatCompletionToolMessageParamContentUnion{
			OfString: openai.String(content),
		}, nil
	}
}

func Chunks[T any](stream *ssestream.Stream[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		for stream.Next() {
			if !yield(stream.Current()) {
				return
			}
		}
	}
}

func ToolResContents(contents []mcp.Content) iter.Seq2[string, mcp.Content] {
	return func(yield func(string, mcp.Content) bool) {
		for _, content := range contents {
			switch content.(type) {
			case *mcp.TextContent:
				if !yield("text", content) {
					return
				}
			case *mcp.AudioContent:
				if !yield("audio", content) {
					return
				}
			case *mcp.ImageContent:
				if !yield("image", content) {
					return
				}
			default:
				if !yield("unknown", content) {
					return
				}
			}
		}
	}
}
