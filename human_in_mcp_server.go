package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// 全局debug开关，通过环境变量 HUMAN_IN_MCP_DEBUG 控制输出
var debugMode = os.Getenv("HUMAN_IN_MCP_DEBUG") == "true"

// debugLog 仅在debug模式下输出日志
func debugLog(format string, v ...interface{}) {
	if debugMode {
		log.Printf(format, v...)
	}
}

type TaskStatus struct {
	TaskId string `json:"taskId"`
	Status string `json:"status"` // pending, processing, completed
	Req    string `json:"req"`    // 原始的请求
	Resp   string `json:"resp"`   // 响应之后携带的summary
}

type TaskManager struct {
	mu    sync.RWMutex
	tasks []*TaskStatus // 使用slice保持添加顺序
}

func NewTaskManager() *TaskManager {
	debugLog("📋 [TaskManager] 初始化任务管理器")
	return &TaskManager{
		tasks: make([]*TaskStatus, 0),
	}
}

func (tm *TaskManager) AddTask(taskId, req string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	// 检查是否已存在（避免重复）
	for _, task := range tm.tasks {
		if task.TaskId == taskId {
			debugLog("⚠️  [TaskManager] 任务已存在，跳过添加 | ID: %s", taskId)
			return
		}
	}
	// 添加新任务到末尾
	tm.tasks = append(tm.tasks, &TaskStatus{
		TaskId: taskId,
		Status: "pending",
		Req:    req,
	})
	debugLog("✅ [TaskManager] 新建任务 | ID: %s | 状态: pending | 请求: %s", taskId, req)
}

func (tm *TaskManager) UpdateTask(taskId, status, resp string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	for _, task := range tm.tasks {
		if task.TaskId == taskId {
			oldStatus := task.Status
			task.Status = status
			task.Resp = resp
			debugLog("🔄 [TaskManager] 更新任务 | ID: %s | %s -> %s | 响应: %s", taskId, oldStatus, status, resp)
			return
		}
	}
	debugLog("⚠️  [TaskManager] 任务不存在，无法更新 | ID: %s", taskId)
}

func (tm *TaskManager) GetTask(taskId string) (*TaskStatus, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	for _, task := range tm.tasks {
		if task.TaskId == taskId {
			return task, true
		}
	}
	return nil, false
}

func (tm *TaskManager) GetAllTasks() []*TaskStatus {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// 直接返回slice的副本，保持添加顺序
	tasks := make([]*TaskStatus, len(tm.tasks))
	copy(tasks, tm.tasks)
	return tasks
}

// DeleteTask 删除指定任务
func (tm *TaskManager) DeleteTask(taskId string) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for i, task := range tm.tasks {
		if task.TaskId == taskId {
			// 删除该任务
			tm.tasks = append(tm.tasks[:i], tm.tasks[i+1:]...)
			debugLog("🗑️  [TaskManager] 删除任务 | ID: %s", taskId)
			return true
		}
	}
	debugLog("⚠️  [TaskManager] 删除任务失败，任务不存在 | ID: %s", taskId)
	return false
}

// UserChoiceResponse 用户的选择响应
type UserChoiceResponse struct {
	TaskId        string `json:"taskId"`        // 任务ID，创建的任务id
	SelectedIndex int    `json:"selectedIndex"` // 用户选择的选项索引（-1表示自定义输入）
	CustomInput   string `json:"customInput"`   // 自定义输入内容
	Continue      bool   `json:"continue"`      // 是否继续对话
}

// RenderTask AI渲染任务，包含需要显示的信息
type RenderTask struct {
	NextOptions  []string `json:"nextOptions"`
	Summary      string   `json:"summary"`
	Difficulties string   `json:"difficulties"`
}

type RenderTaskStatusful struct {
	RenderTask
	Status       string `json:"status"`
	ActualChoice string `json:"actualChoice"` // 使用用户选择的Req记录
}

// SessionManager 全局单例会话管理器
type SessionManager struct {
	Out         chan UserChoiceResponse // 用户响应通道
	Render      chan RenderTask         // AI渲染任务通道（用于web端显示）
	mu          sync.RWMutex            // 保护responses切片
	responses   []UserChoiceResponse    // 缓存已接收的响应
	renderTasks []RenderTask            // 缓存AI渲染任务

	//=====  -- 所有开放的对象都等于SessionManager的相关调用
	Taskmng *TaskManager // 任务管理器

}

// 全局单例
var globalSessionManager = &SessionManager{
	Out:         make(chan UserChoiceResponse, 10),
	Render:      make(chan RenderTask, 10),
	responses:   make([]UserChoiceResponse, 0, 10),
	renderTasks: make([]RenderTask, 0, 10),
	Taskmng:     NewTaskManager(),
}

// AddResponse 添加响应到队列
func (sm *SessionManager) AddResponse(resp UserChoiceResponse) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.responses = append(sm.responses, resp)
	debugLog("📥 [SessionManager] 添加响应到队列 | TaskID: %s | 输入: %s", resp.TaskId, resp.CustomInput)
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
	debugLog("📤 [SessionManager] 添加AI渲染任务 | 摘要: %s | 困难: %s", task.Summary, task.Difficulties)
}

// GetRenderTasks 获取所有AI渲染任务
func (sm *SessionManager) GetRenderTasks() []RenderTask {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.renderTasks
}

// RemoveFirstRenderTask 移除第一个渲染任务（已处理）
func (sm *SessionManager) RemoveFirstRenderTask() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if len(sm.renderTasks) > 0 {
		removed := sm.renderTasks[0]
		sm.renderTasks = sm.renderTasks[1:]
		debugLog("🗑️  [SessionManager] 移除已处理的渲染任务 | 摘要: %s", removed.Summary)
	}
}

// 通过队列来维护存储 chan自己不支持队列方式的查询和存储
// PushResponse 发送响应到Out通道
func (sm *SessionManager) PushResponse(resp UserChoiceResponse) {
	resp.TaskId = insIdGen() // 生成唯一任务ID
	sm.AddResponse(resp)

	sm.Taskmng.AddTask(resp.TaskId, resp.CustomInput) // 将任务添加到任务管理器

	select {
	case sm.Out <- resp:
		debugLog("📨 [SessionManager] 响应已发送到Out通道 | TaskID: %s | 继续: %t", resp.TaskId, resp.Continue)
	default:
		debugLog("⚠️  [SessionManager] Out通道已满，响应未发送 | TaskID: %s", resp.TaskId)
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
• 开始的工作初始化的时候 直接调用这个工具进行询问,可以提供一些元信息的可选项,比如输出当前目录,测试网络连通性等基础功能

注意事项：
• 这是一个持续循环，直到用户明确选择结束
• 收到返回结果后，务必按照"【重要指令】"执行`),
		mcp.WithString("summary", mcp.Required(), mcp.Description("完成任务的简单总结,如果处于起步或初始化状态,直接传递当前工作目录地址即可")),
		mcp.WithString("taskId", mcp.Description("插件内部提供的唯一任务Id,必须通过该系统内部进行指定,对于完成的每个任务都会生成一个唯一的任务Id , 如果没有对话历史或处于起步或初始化状态,传值不做要求")),

		mcp.WithString("difficulties", mcp.Required(), mcp.Description("遇到的困难、需要的帮助或其他重要信息")),
		mcp.WithString("nextOptions", mcp.Required(),
			mcp.Description("接下来的任务可选项，JSON数组字符串格式，例如: [\"继续优化代码\", \"添加测试\", \"提交代码\", \"结束\"]")),
	)
}

func process(sm *SessionManager, id, summary string) {
	if id != "" {
		debugLog("🎯 [MCP] 处理任务完成 | TaskID: %s | 摘要: %s", id, summary)
		sm.Taskmng.UpdateTask(id, "completed", summary)
	}
}

// humanInteractionHandler 处理人机交互请求
func humanInteractionHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	startTime := time.Now()
	debugLog("🤖 [MCP] ========== 人机交互请求开始 ==========")

	// 解析参数
	summary, _ := req.RequireString("summary")
	difficulties, _ := req.RequireString("difficulties")
	nextOptionsStr, _ := req.RequireString("nextOptions")
	id, _ := req.RequireString("taskId")

	debugLog("📝 [MCP] 请求参数 | TaskID: %s | 摘要: %s | 困难: %s", id, summary, difficulties)

	process(globalSessionManager, id, summary)

	var nextOptions []string
	if err := json.Unmarshal([]byte(nextOptionsStr), &nextOptions); err != nil {
		nextOptions = []string{nextOptionsStr}
	}
	debugLog("📋 [MCP] 下一步选项: %v", nextOptions)

	// 创建渲染任务并发送到Render通道（供web端显示）
	renderTask := RenderTask{
		NextOptions:  nextOptions,
		Summary:      summary,
		Difficulties: difficulties,
	}
	globalSessionManager.AddRenderTask(renderTask)
	select {
	case globalSessionManager.Render <- renderTask:
		debugLog("📤 [MCP] 渲染任务已发送到Render通道")
	default:
		debugLog("⚠️  [MCP] Render通道已满")
	}

	// 阻塞等待用户响应
	debugLog("⏳ [MCP] 等待用户响应...")
	response := <-globalSessionManager.Out
	debugLog("✅ [MCP] 收到用户响应 | TaskID: %s | 输入: %s | 继续: %t", response.TaskId, response.CustomInput, response.Continue)

	globalSessionManager.Taskmng.UpdateTask(response.TaskId, "processing", summary) // 更新任务状态为processing

	duration := time.Since(startTime)
	debugLog("⏱️  [MCP] 人机交互请求处理完成 | 耗时: %v", duration)
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
 • taskId 请在完成之后,调用human_interaction工具的时候携带这个taskId: %s ,以便追踪和管理任务状态
请记住：这是持续对话循环，每次完成任务后都要调用 human_interaction 工具！`,
			response.CustomInput,
			response.TaskId,
		)
	} else {
		aiPrompt = `【对话结束】
用户选择结束本次对话。
请停止工作，不需要再调用任何工具。`
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
