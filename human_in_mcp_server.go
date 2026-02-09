package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// UserChoiceResponse 用户的选择响应
type UserChoiceResponse struct {
	ConversationID string `json:"conversationId"` // 对话ID
	SelectedIndex  int    `json:"selectedIndex"`  // 用户选择的选项索引（-1表示自定义输入）
	CustomInput    string `json:"customInput"`    // 自定义输入内容
	Continue       bool   `json:"continue"`       // 是否继续对话
}

// SessionManager 管理单次工具调用的交互会话
type SessionManager struct {
	toRender       chan struct{}           // 触发渲染（可选，此处简化为直接调用）
	Out            chan UserChoiceResponse // 返回给 handler 的最终有效响应
	nextOptions    []string
	conversationID string
	summary        string
	difficulties   string
}

var globalReader = bufio.NewReader(os.Stdin)
var sessionMutex sync.Mutex // 保证同一时间只有一个会话（简化设计）

// render 启动交互循环，直到获得有效输入
func (sm *SessionManager) render() {
	for {
		// ========== 显示界面给用户 ==========
		fmt.Println("\n" + strings.Repeat("=", 70))
		fmt.Printf("🤖 AI 任务完成报告 [对话ID: %s]\n", sm.conversationID)
		fmt.Println(strings.Repeat("=", 70))
		fmt.Println("\n📋 任务总结:")
		fmt.Println(sm.summary)
		if sm.difficulties != "" && sm.difficulties != "无" && sm.difficulties != "无困难" {
			fmt.Println("\n⚠️ 遇到的问题/需要的帮助:")
			fmt.Println(sm.difficulties)
		}
		fmt.Println("\n🔄 接下来的可选项:")
		for i, option := range sm.nextOptions {
			fmt.Printf(" [%d] %s\n", i+1, option)
		}
		fmt.Println(" [0] 自定义输入")
		fmt.Println(" [q] 结束对话")
		fmt.Println("\n" + strings.Repeat("-", 70))

		// ========== 获取用户输入 ==========
		fmt.Print("\n请选择操作 (输入数字或命令): ")
		input, err := globalReader.ReadString('\n')
		if err != nil {
			fmt.Printf("⚠️ 输入错误，请重试: %v\n", err)
			continue
		}
		input = strings.TrimSpace(input)

		// ========== 处理并验证用户输入 ==========
		response := UserChoiceResponse{
			ConversationID: sm.conversationID,
		}

		switch input {
		case "q", "Q", "quit", "exit":
			response.Continue = false
			response.CustomInput = "用户选择结束对话"
			sm.Out <- response
			return

		case "0":
			fmt.Print("\n请输入您的指示: ")
			customInput, err := globalReader.ReadString('\n')
			if err != nil {
				fmt.Printf("⚠️ 自定义输入读取失败，请重试: %v\n", err)
				continue
			}
			response.Continue = true
			response.CustomInput = strings.TrimSpace(customInput)
			response.SelectedIndex = -1
			sm.Out <- response
			return

		default:
			var index int
			_, err := fmt.Sscanf(input, "%d", &index)
			if err != nil || index < 1 || index > len(sm.nextOptions) {
				fmt.Printf("❌ 无效输入！请输入 0-%d 之间的数字，或 q 退出。\n", len(sm.nextOptions))
				continue // 重试
			}
			response.Continue = true
			response.SelectedIndex = index - 1
			response.CustomInput = sm.nextOptions[index-1]
			sm.Out <- response
			return
		}
	}
}

// HumanInTool 定义 MCP 工具
func HumanInTool() mcp.Tool {
	return mcp.NewTool(
		"human_interaction",
		mcp.WithDescription(`【重要：人机交互循环工具】
用途：AI完成每个任务后，必须调用此工具向用户展示结果并获取下一步指示。
工作流程（无限循环）：
1. AI完成用户指派的任务
2. AI调用此工具展示任务总结
3. 用户查看结果并选择下一步
4. AI收到用户的新任务指示
5. 重复步骤1...
调用时机：
• 每次完成任务后
• 需要用户决策时
• 需要展示中间结果时
注意事项：
• 必须保持相同的 conversationId 以维持对话上下文
• 这是一个持续循环，直到用户明确选择结束
• 收到返回结果后，务必按照"【重要指令】"执行`),
		mcp.WithString("summary", mcp.Required(), mcp.Description("完成任务的简单总结")),
		mcp.WithString("difficulties", mcp.Required(), mcp.Description("遇到的困难、需要的帮助或其他重要信息")),
		mcp.WithString("conversationId", mcp.Required(), mcp.Description("对话ID，用于跟踪多轮对话，必须保持一致，可使用时间戳或UUID")),
		mcp.WithString("nextOptions", mcp.Required(),
			mcp.Description("接下来的任务可选项，JSON数组字符串格式，例如: [\"继续优化代码\", \"添加测试\", \"提交代码\", \"结束\"]")),
	)
}

// humanInteractionHandler 处理人机交互请求
func humanInteractionHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	// 解析参数
	summary, _ := req.RequireString("summary")
	difficulties, _ := req.RequireString("difficulties")
	conversationID, _ := req.RequireString("conversationId")
	nextOptionsStr, _ := req.RequireString("nextOptions")

	var nextOptions []string
	if err := json.Unmarshal([]byte(nextOptionsStr), &nextOptions); err != nil {
		nextOptions = []string{nextOptionsStr}
	}

	// 创建会话管理器
	sm := &SessionManager{
		Out:            make(chan UserChoiceResponse, 1),
		nextOptions:    nextOptions,
		conversationID: conversationID,
		summary:        summary,
		difficulties:   difficulties,
	}

	// 启动渲染和输入循环（goroutine）
	go sm.render()

	// 阻塞等待用户有效响应
	response := <-sm.Out

	// ========== 构建返回结果 ==========
	var aiPrompt string
	if response.Continue {
		aiPrompt = fmt.Sprintf(`【用户任务】
%s
【重要指令】
1. 请立即执行上述用户任务
2. 完成任务后，必须再次调用 human_interaction 工具向用户展示结果
3. 调用时使用相同的 conversationId: %s
4. 调用参数：
 • summary: 你完成任务的总结
 • difficulties: 遇到的问题或困难
 • conversationId: %s
 • nextOptions: 建议的下一步选项（JSON数组格式）
【对话上下文】
• 对话ID: %s
• 当前是第 %d 轮交互
请记住：这是持续对话循环，每次完成任务后都要调用 human_interaction 工具！`,
			response.CustomInput,
			conversationID,
			conversationID,
			conversationID,
			1, // TODO: 可扩展为计数器
		)
		fmt.Printf("\n✅ 已记录您的选择，将指示AI执行: %s\n", response.CustomInput)
	} else {
		aiPrompt = fmt.Sprintf(`【对话结束】
用户选择结束本次对话。
对话ID: %s
结束原因: %s
请停止工作，不需要再调用任何工具。`, conversationID, response.CustomInput)
		fmt.Printf("\n👋 对话已结束\n")
	}
	fmt.Println(strings.Repeat("=", 70) + "\n")

	// 返回结构化结果 + AI提示
	jsonData, _ := json.MarshalIndent(response, "", "  ")
	return mcp.NewToolResultText(fmt.Sprintf("%s\n\n---\n\n用户响应数据（JSON）:\n%s",
		aiPrompt,
		string(jsonData),
	)), nil
}

// main 启动 MCP 服务器
func main() {
	mcpServer := server.NewMCPServer("human-in-mcp", "v1.0.0",
		server.WithToolCapabilities(true),
	)
	mcpServer.AddTool(HumanInTool(), humanInteractionHandler)
	sseServer := server.NewSSEServer(mcpServer)

	mux := http.NewServeMux()
	mux.Handle("/", sseServer)
	fmt.Println("✅ Human-In-MCP Server running on http://localhost:8093")
	fmt.Println("📝 功能: AI任务完成后的人机交互循环")
	fmt.Println("🔧 工具: human_interaction")
	if err := http.ListenAndServe("localhost:8093", mux); err != nil {
		panic(err)
	}
}
