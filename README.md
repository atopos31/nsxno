# NSXNO

NSXNO 是一个基于 Go 语言的项目，实现了一个能够通过 OpenAI API 与模型上下文协议（MCP）工具交互的 AI 代理。该代理可以处理自然语言查询并相应地执行工具，支持流式响应和并行工具调用。

## 功能特性

- 集成 OpenAI 的 GPT 模型
- 支持模型上下文协议（MCP）工具
- 流式响应实现实时输出
- 并行工具执行能力
- 可扩展的代理架构

## 环境要求

- Go 1.24.3 或更高版本
- 访问兼容 OpenAI 的 API 端点
- MCP 服务器端点

## 依赖项

- `github.com/modelcontextprotocol/go-sdk v0.2.0`
- `github.com/openai/openai-go v1.11.0`
- `github.com/tidwall/gjson v1.18.0`

## 安装

1. 克隆仓库：
   ```bash
   git clone https://github.com/atopos31/nsxno.git
   cd nsxno
   ```

2. 安装依赖：
   ```bash
   go mod tidy
   ```

## 配置

应用程序需要设置几个环境变量：

- `OPENAI_BASE_URL`: OpenAI API 的基础 URL
- `OPENAI_API_KEY`: 你的 OpenAI API 密钥
- `OPENAI_MODEL`: 要使用的模型（例如 gpt-4）
- `TEST_MCP_BASE_URL`: MCP 服务器端点

示例：
```bash
export OPENAI_BASE_URL="https://api.openai.com/v1"
export OPENAI_API_KEY="your-api-key-here"
export OPENAI_MODEL="gpt-4"
export TEST_MCP_BASE_URL="http://localhost:3000/mcp"
```

## 使用方法

运行应用程序：

```bash
go run main.go
```

默认情况下，应用程序会让 AI 执行 "分两次获取一下198.199.77.16和115.190.78.97的ip归属地 两次之间分别回复我"（分两步获取 198.199.77.16 和 115.190.78.97 的 IP 归属地，并在两次操作之间进行回复）。

## 架构设计

项目由两个主要组件构成：

1. **主应用程序 (`main.go`)**: 设置与 OpenAI API 和 MCP 服务器的连接，配置代理并运行交互循环。

2. **代理实现 (`react/agent.go`)**: 包含处理 AI 响应、处理工具调用和管理对话流程的核心逻辑。

### 核心组件

- `Agent.Run()`: 处理与 AI 对话的主执行循环
- `McpToolHandler()`: 通过与 MCP 服务器通信来处理工具调用
- `ToolsFormMCP()`: 将 MCP 工具转换为 OpenAI 工具格式
- 支持流式响应处理和实时输出

## 贡献

欢迎提交贡献！请随时提交 Pull Request。

## 许可证

本项目采用 MIT 许可证 - 详情请见 LICENSE 文件。