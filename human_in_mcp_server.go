package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

// RenderTask AI渲染任务，包含需要显示的信息
type RenderTask struct {
	NextOptions    []string `json:"nextOptions"`
	ConversationID string   `json:"conversationId"`
	Summary        string   `json:"summary"`
	Difficulties   string   `json:"difficulties"`
}

// SessionManager 全局单例会话管理器
type SessionManager struct {
	Out            chan UserChoiceResponse // 用户响应通道
	Render         chan RenderTask         // AI渲染任务通道（用于web端显示）
	mu             sync.RWMutex            // 保护responses切片
	responses      []UserChoiceResponse    // 缓存已接收的响应
	renderTasks    []RenderTask            // 缓存AI渲染任务
	processedTasks []ProcessedTask         // 已处理的任务
}

// 全局单例
var globalSessionManager = &SessionManager{
	Out:            make(chan UserChoiceResponse, 10),
	Render:         make(chan RenderTask, 10),
	responses:      make([]UserChoiceResponse, 0, 10),
	renderTasks:    make([]RenderTask, 0, 10),
	processedTasks: make([]ProcessedTask, 0, 50),
}

// AddResponse 添加响应到队列
func (sm *SessionManager) AddResponse(resp UserChoiceResponse) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.responses = append(sm.responses, resp)
}

// GetResponses 获取所有响应
func (sm *SessionManager) GetResponses() []UserChoiceResponse {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.responses
}

// AddRenderTask 添加AI渲染任务
func (sm *SessionManager) AddRenderTask(task RenderTask) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.renderTasks = append(sm.renderTasks, task)
}

// GetRenderTasks 获取所有AI渲染任务
func (sm *SessionManager) GetRenderTasks() []RenderTask {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.renderTasks
}

// PushResponse 发送响应到Out通道
func (sm *SessionManager) PushResponse(resp UserChoiceResponse) {
	sm.AddResponse(resp)
	select {
	case sm.Out <- resp:
	default:
	}
}

// AddProcessedTask 添加已处理任务
func (sm *SessionManager) AddProcessedTask(task ProcessedTask) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.processedTasks = append(sm.processedTasks, task)
}

// GetProcessedTasks 获取已处理任务列表
func (sm *SessionManager) GetProcessedTasks() []ProcessedTask {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.processedTasks
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

	var nextOptions []string
	if err := json.Unmarshal([]byte(nextOptionsStr), &nextOptions); err != nil {
		nextOptions = []string{nextOptionsStr}
	}

	// 创建渲染任务并发送到Render通道（供web端显示）
	renderTask := RenderTask{
		NextOptions:    nextOptions,
		ConversationID: conversationID,
		Summary:        summary,
		Difficulties:   difficulties,
	}
	globalSessionManager.AddRenderTask(renderTask)
	select {
	case globalSessionManager.Render <- renderTask:
	default:
	}

	// 阻塞等待用户响应
	response := <-globalSessionManager.Out

	// 构建返回结果
	var aiPrompt string
	if response.Continue {
		aiPrompt = fmt.Sprintf(`【用户任务】
%s

【重要指令】
1. 请立即执行上述用户任务
2. 完成任务后，必须再次调用 human_interaction 工具向用户展示结果
3. 调用参数：
 • summary: 你完成任务的总结
 • difficulties: 遇到的问题或困难
 • nextOptions: 建议的下一步选项（JSON数组格式）

请记住：这是持续对话循环，每次完成任务后都要调用 human_interaction 工具！`,
			response.CustomInput,
		)
	} else {
		aiPrompt = fmt.Sprintf(`【对话结束】
用户选择结束本次对话。
请停止工作，不需要再调用任何工具。`)
	}

	// 返回结构化结果 + AI提示
	jsonData, _ := json.MarshalIndent(response, "", "  ")
	return mcp.NewToolResultText(fmt.Sprintf("%s\n\n---\n\n用户响应数据（JSON）:\n%s",
		aiPrompt,
		string(jsonData),
	)), nil
}

// main 启动 MCP 服务器
func main() {
	// 启动任务管理HTTP服务器
	StartTaskServer()

	mcpServer := server.NewMCPServer("human-in-mcp", "v1.0.0",
		server.WithToolCapabilities(true),
	)
	mcpServer.AddTool(HumanInTool(), humanInteractionHandler)
	sseServer := server.NewSSEServer(mcpServer)

	mux := http.NewServeMux()
	mux.Handle("/", sseServer)
	fmt.Println("✅ Human-In-MCP Server running on http://localhost:8093")
	fmt.Println("📝 任务管理页面: http://localhost:8094")
	if err := http.ListenAndServe("localhost:8093", mux); err != nil {
		panic(err)
	}
}
