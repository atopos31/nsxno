package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/atopos31/nsxno/mcpconv"
	"github.com/atopos31/nsxno/react"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func main() {
	ctx := context.Background()
	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "mcp-client", Version: "v1.0.0"}, nil)
	transport := mcp.NewSSEClientTransport(os.Getenv("TEST_MCP_BASE_URL"), nil)
	session, err := mcpClient.Connect(ctx, transport)
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()
	tools, err := mcpconv.ToolsFormMCP(ctx, session)
	if err != nil {
		log.Fatal(err)
	}
	client := openai.NewClient(
		option.WithBaseURL(os.Getenv("OPENAI_BASE_URL")),
		option.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
	)
	agent := react.New(client, 20)
	question := "分两次获取一下198.199.77.16和115.190.78.97的ip归属地 两次之间分别回复我总结信息"
	model := os.Getenv("OPENAI_MODEL")

	for content, err := range agent.RunStream(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(question),
		},
		Tools: tools,
		Model: model,
		// 开启并行工具调用
		ParallelToolCalls: openai.Bool(true),
	}, react.McpToolHandler(session)) {
		if err != nil {
			slog.Error("react", "err", err)
		}
		switch content.Cate {
		case "message":
			if len(content.Chunk.Choices) > 0 {
				fmt.Print(content.Chunk.Choices[0].Delta.Content)
			}
		case "toolcall":
			fmt.Printf("\ntoolcall: step=%d id=%s name=%s args=%s\n", content.Step, content.ToolCall.ID, content.ToolCall.Function.Name, content.ToolCall.Function.Arguments)
		case "toolres":
			fmt.Printf("\ntoolres: step=%d id=%s content=%s\n", content.Step, content.ToolResID, content.ToolRes.OfString.Value)
		}
	}
	fmt.Println("")
}
