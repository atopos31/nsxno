package mcpconv

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/openai/openai-go"
)

// 将 Mcp 的工具列表转换成 OpenAI 的工具列表
func ToolsFormMCP(ctx context.Context, session *mcp.ClientSession) ([]openai.ChatCompletionToolParam, error) {
	mcpToolsRes, err := session.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}
	var tools []openai.ChatCompletionToolParam
	for _, tool := range mcpToolsRes.Tools {
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
