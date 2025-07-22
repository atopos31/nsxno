package main

import (
	"context"
	"encoding/json"
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
	mcpTools, err := session.ListTools(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	tools, err := ToolsFormMCP(mcpTools)
	if err != nil {
		log.Fatal(err)
	}
	client := openai.NewClient(
		option.WithBaseURL(os.Getenv("OPENAI_BASE_URL")),
		option.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
	)
	agent := react.New(client, 20)
	question := "分两次获取一下198.199.77.16和115.190.78.97的ip归属地 两次之间分别回复我"
	model := "qwen-3"

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
	close(reactChunk)
	// 等待打印出所有响应回复
	wg.Wait()
	fmt.Printf("\n")
}

// 将 Mcp 的工具列表转换成 OpenAI 的工具列表
func ToolsFormMCP(list *mcp.ListToolsResult) ([]openai.ChatCompletionToolParam, error) {
	var tools []openai.ChatCompletionToolParam
	for _, tool := range list.Tools {
		var toolParam openai.ChatCompletionToolParam
		data, err := json.Marshal(tool.InputSchema)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &toolParam.Function.Parameters); err != nil {
			return nil, err
		}
		toolParam.Function.Name = tool.Name
		toolParam.Function.Description = openai.String(tool.Description)
		toolParam.Function.Strict = openai.Bool(true)
		toolParam.Function.Parameters["additionalProperties"] = false
		tools = append(tools, toolParam)
	}
	return tools, nil
}
