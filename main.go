package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

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
	tools, err := react.ToolsFormMCP(ctx, session)
	if err != nil {
		log.Fatal(err)
	}
	client := openai.NewClient(
		option.WithBaseURL(os.Getenv("OPENAI_BASE_URL")),
		option.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
	)
	agent := react.New(client, 20)
	question := "分两次获取一下198.199.77.16和115.190.78.97的ip归属地 两次之间分别回复我"
	model := os.Getenv("OPENAI_MODEL")

	reactChunk := make(chan react.ReactChunk, 10)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for chunk := range reactChunk {
			switch chunk.Type {
			case "text":
				fmt.Print(chunk.Content)
			case "toolcall":
				fmt.Printf("\ntoolcall: step=%d name=%s args=%s\n", chunk.Step, chunk.ToolCall.Function.Name, chunk.ToolCall.Function.Arguments)
			case "toolres":
				fmt.Printf("\ntoolres: step=%d name=%s content=%s\n", chunk.Step, chunk.Type, chunk.ToolRes.OfString.Value)
			}
		}
	}()
	_, err = agent.Run(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(question),
		},
		Tools: tools,
		Seed:  openai.Int(100),
		Model: model,
		// 开启并行工具调用
		ParallelToolCalls: openai.Bool(true),
	}, react.McpToolHandler(session), reactChunk)
	if err != nil {
		panic(err)
	}
	// 等待打印出所有响应回复
	wg.Wait()
	fmt.Printf("\n")
}
