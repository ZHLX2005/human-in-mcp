package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// HumanInMCP 实现人机交互循环
// AI 完成任务后通过此工具向用户展示信息并获取下一步指示

// TaskSummaryRequest AI 发送给用户的任务总结请求
type TaskSummaryRequest struct {
	Summary        string   `json:"summary"`        // 完成任务的简单总结
	Difficulties   string   `json:"difficulties"`   // 遇到的困难或需要的帮助
	NextOptions    []string `json:"nextOptions"`    // 接下来的任务可选项
	ConversationID string   `json:"conversationId"` // 对话ID（用于跟踪多轮对话）
}

// UserChoiceResponse 用户的选择响应
type UserChoiceResponse struct {
	ConversationID string `json:"conversationId"` // 对话ID
	SelectedIndex  int    `json:"selectedIndex"`  // 用户选择的选项索引（-1表示自定义输入）
	CustomInput    string `json:"customInput"`    // 自定义输入内容
	Continue       bool   `json:"continue"`       // 是否继续对话
}

// 全局 reader 用于读取用户输入
var reader *bufio.Reader

func init() {
	reader = bufio.NewReader(os.Stdin)
}

// HumanInTool 定义 MCP 工具
func HumanInTool() mcp.Tool {
	return mcp.NewTool(
		"human_interaction",
		mcp.WithDescription("AI完成任务后向用户展示信息并获取下一步指示。实现人机交互循环，支持多轮对话。"),
		mcp.WithString("summary", mcp.Required(), mcp.Description("完成任务的简单总结")),
		mcp.WithString("difficulties", mcp.Required(), mcp.Description("遇到的困难、需要的帮助或其他重要信息")),
		mcp.WithString("conversationId", mcp.Required(), mcp.Description("对话ID，用于跟踪多轮对话，可使用时间戳或UUID")),
		mcp.WithString("nextOptions", mcp.Required(),
			mcp.Description("接下来的任务可选项，JSON数组字符串格式，例如: [\"继续优化代码\", \"添加测试\", \"提交代码\", \"结束\"]")),
	)
}

// humanInteractionHandler 处理人机交互请求
func humanInteractionHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 解析参数
	summary, _ := req.RequireString("summary")
	difficulties, _ := req.RequireString("difficulties")
	conversationID, _ := req.RequireString("conversationId")
	nextOptionsStr, _ := req.RequireString("nextOptions")

	// 解析选项列表
	var nextOptions []string
	if err := json.Unmarshal([]byte(nextOptionsStr), &nextOptions); err != nil {
		nextOptions = []string{nextOptionsStr} // 如果解析失败，将字符串作为单个选项
	}

	// ========== 显示界面给用户 ==========
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Printf("🤖 AI 任务完成报告 [对话ID: %s]\n", conversationID)
	fmt.Println(strings.Repeat("=", 70))

	fmt.Println("\n📋 任务总结:")
	fmt.Println(summary)

	if difficulties != "" && difficulties != "无" && difficulties != "无困难" {
		fmt.Println("\n⚠️  遇到的问题/需要的帮助:")
		fmt.Println(difficulties)
	}

	fmt.Println("\n🔄 接下来的可选项:")
	for i, option := range nextOptions {
		fmt.Printf("  [%d] %s\n", i+1, option)
	}
	fmt.Println("  [0] 自定义输入")
	fmt.Println("  [q] 结束对话")

	fmt.Println("\n" + strings.Repeat("-", 70))

	// ========== 获取用户输入 ==========
	fmt.Print("\n请选择操作 (输入数字或命令): ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return mcp.NewToolResultError("读取用户输入失败: " + err.Error()), nil
	}

	input = strings.TrimSpace(input)

	// ========== 处理用户输入 ==========
	response := UserChoiceResponse{
		ConversationID: conversationID,
	}

	switch input {
	case "q", "Q", "quit", "exit":
		response.Continue = false
		response.CustomInput = "用户选择结束对话"

	case "0":
		// 自定义输入
		fmt.Print("\n请输入您的指示: ")
		customInput, err := reader.ReadString('\n')
		if err != nil {
			return mcp.NewToolResultError("读取自定义输入失败: " + err.Error()), nil
		}
		response.Continue = true
		response.CustomInput = strings.TrimSpace(customInput)
		response.SelectedIndex = -1

	default:
		// 处理数字选择
		var index int
		_, err := fmt.Sscanf(input, "%d", &index)
		if err != nil || index < 1 || index > len(nextOptions) {
			return mcp.NewToolResultError(fmt.Sprintf("无效的选择，请输入 0-%d 之间的数字或 q 退出", len(nextOptions))), nil
		}
		response.Continue = true
		response.SelectedIndex = index - 1
		response.CustomInput = nextOptions[index-1]
	}

	// ========== 构建返回结果 ==========
	resultJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("构建响应失败: " + err.Error()), nil
	}

	// 显示确认信息
	if response.Continue {
		fmt.Printf("\n✅ 已记录您的选择，将继续执行...\n")
	} else {
		fmt.Printf("\n👋 对话已结束\n")
	}
	fmt.Println(strings.Repeat("=", 70) + "\n")

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// main 启动 MCP 服务器
func main() {
	mcpServer := server.NewMCPServer("human-in-mcp", "v1.0.0",
		server.WithToolCapabilities(true),
	)

	mcpServer.AddTool(HumanInTool(), humanInteractionHandler)

	sseServer := server.NewSSEServer(mcpServer)

	// 使用 ServeMux 来正确路由请求
	mux := http.NewServeMux()
	mux.Handle("/", sseServer)

	fmt.Println("✅ Human-In-MCP Server running on http://localhost:8093")
	fmt.Println("📝 功能: AI任务完成后的人机交互循环")
	fmt.Println("🔧 工具: human_interaction")
	if err := http.ListenAndServe("localhost:8093", mux); err != nil {
		panic(err)
	}
}
