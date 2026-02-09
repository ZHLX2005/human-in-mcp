package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// TaskRequest 任务请求结构
type TaskRequest struct {
	CustomInput   string `json:"customInput"`
	Continue      bool   `json:"continue"`
	SelectedIndex *int   `json:"selectedIndex"` // 可选，从AI选项中选择
}

// 启动HTTP服务器
func StartTaskServer() {
	// API路由
	http.HandleFunc("/", serveHomePage)
	http.HandleFunc("/api/tasks", handleTasks)
	http.HandleFunc("/api/tasks/list", handleListTasks)
	http.HandleFunc("/api/tasks/status", handleTaskStatus) // 获取任务状态
	http.HandleFunc("/api/tasks/delete", handleDeleteTask) // 删除任务
	http.HandleFunc("/api/tasks/clear", handleClearTasks) // 清空所有任务
	http.HandleFunc("/api/render-tasks", handleRenderTasks)
	http.HandleFunc("/api/render-tasks/select", handleSelectRenderTask)
	http.HandleFunc("/api/render-tasks/abandon", handleAbandonRenderTask) // 遗弃AI渲染任务
	http.HandleFunc("/api/format/get", handleGetFormat)                   // 获取格式化字符串
	http.HandleFunc("/api/format/set", handleSetFormat)                   // 设置格式化字符串

	fmt.Println("📝 任务管理页面: http://localhost:8094")
	go http.ListenAndServe(":8094", nil)
}

// serveHomePage 提供主页HTML
func serveHomePage(w http.ResponseWriter, r *http.Request) {
	// 读取HTML文件
	htmlPath := "templates/index.html"
	content, err := os.ReadFile(htmlPath)
	if err != nil {
		debugLog("❌ [HTTP] 读取HTML文件失败 | %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(content)
}

// handleTasks 处理手动任务添加请求
func handleTasks(w http.ResponseWriter, r *http.Request) {
	debugLog("🌐 [HTTP] %s %s | 处理手动任务添加请求", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		debugLog("❌ [HTTP] 方法不允许 | %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var task TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		debugLog("❌ [HTTP] 请求体解析失败 | %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 验证必填字段
	if task.CustomInput == "" {
		debugLog("❌ [HTTP] 缺少必填字段 | customInput")
		http.Error(w, "customInput is required", http.StatusBadRequest)
		return
	}

	// 创建响应并添加到队列
	response := UserChoiceResponse{
		CustomInput:   task.CustomInput,
		Continue:      task.Continue,
		SelectedIndex: -1,
	}

	globalSessionManager.PushResponse(response)
	debugLog("✅ [HTTP] 手动任务已添加 | 输入: %s | 继续: %t", task.CustomInput, task.Continue)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Task added to queue",
	})
}

// handleListTasks 返回当前待处理的任务列表（pending状态）
func handleListTasks(w http.ResponseWriter, r *http.Request) {
	debugLog("🌐 [HTTP] %s %s | 获取待处理任务列表", r.Method, r.URL.Path)

	// 获取所有任务状态
	allTasks := globalSessionManager.Taskmng.GetAllTasks()

	// 筛选出pending状态的任务
	pendingTasks := make([]*TaskStatus, 0)
	for _, task := range allTasks {
		if task.Status == "pending" {
			pendingTasks = append(pendingTasks, task)
		}
	}

	debugLog("📊 [HTTP] 返回待处理任务列表 | 数量: %d", len(pendingTasks))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pendingTasks)
}

// handleRenderTasks 返回AI渲染任务列表
func handleRenderTasks(w http.ResponseWriter, r *http.Request) {
	debugLog("🌐 [HTTP] %s %s | 获取AI渲染任务", r.Method, r.URL.Path)
	tasks := globalSessionManager.GetRenderTasks()
	debugLog("📊 [HTTP] 返回AI渲染任务 | 数量: %d", len(tasks))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// handleSelectRenderTask 处理从AI渲染任务中选择选项
func handleSelectRenderTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SelectedIndex *int   `json:"selectedIndex"`
		CustomInput    string `json:"customInput"`
		Continue       bool   `json:"continue"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 获取第一个渲染任务
	renderTasks := globalSessionManager.GetRenderTasks()
	if len(renderTasks) == 0 {
		http.Error(w, "No render task available", http.StatusNotFound)
		return
	}
	targetTask := renderTasks[0]

	// 创建响应
	response := UserChoiceResponse{
		Continue:     req.Continue,
		SelectedIndex: -1,
	}

	var responseText string
	if req.SelectedIndex != nil && *req.SelectedIndex >= 0 && *req.SelectedIndex < len(targetTask.NextOptions) {
		response.SelectedIndex = *req.SelectedIndex
		responseText = targetTask.NextOptions[*req.SelectedIndex]
	} else if req.CustomInput != "" {
		responseText = req.CustomInput
	} else {
		responseText = "结束对话"
	}
	response.CustomInput = responseText

	// 发送到Out通道
	globalSessionManager.PushResponse(response)

	// 如果是结束对话，直接标记任务为完成（因为AI不会再给反馈）
	if !req.Continue {
		// 获取刚创建的任务（最后一个任务）
		allTasks := globalSessionManager.Taskmng.GetAllTasks()
		if len(allTasks) > 0 {
			lastTask := allTasks[len(allTasks)-1]
			globalSessionManager.Taskmng.UpdateTask(lastTask.TaskId, "completed", "用户结束对话")
			debugLog("✅ [HTTP] 结束任务已直接标记为完成 | TaskID: %s", lastTask.TaskId)
		}
	}

	// 移除已处理的渲染任务
	globalSessionManager.RemoveFirstRenderTask()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Response sent",
	})
}

// handleAbandonRenderTask 遗弃AI渲染任务
func handleAbandonRenderTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	debugLog("🌐 [HTTP] %s %s | 遗弃AI渲染任务", r.Method, r.URL.Path)

	// 获取第一个渲染任务
	renderTasks := globalSessionManager.GetRenderTasks()
	if len(renderTasks) == 0 {
		debugLog("❌ [HTTP] 没有可遗弃的渲染任务")
		http.Error(w, "No render task available", http.StatusNotFound)
		return
	}

	abandonedTask := renderTasks[0]
	debugLog("🗑️  [HTTP] 遗弃AI渲染任务 | 摘要: %s", abandonedTask.Summary)

	// 移除第一个渲染任务
	globalSessionManager.RemoveFirstRenderTask()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Task abandoned",
	})
}

// handleDeleteTask 删除指定任务
func handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	debugLog("🌐 [HTTP] %s %s | 删除任务", r.Method, r.URL.Path)

	var req struct {
		TaskId string `json:"taskId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		debugLog("❌ [HTTP] 请求体解析失败 | %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.TaskId == "" {
		debugLog("❌ [HTTP] 缺少必填字段 | taskId")
		http.Error(w, "taskId is required", http.StatusBadRequest)
		return
	}

	// 删除任务
	if globalSessionManager.Taskmng.DeleteTask(req.TaskId) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": "Task deleted",
		})
	} else {
		http.Error(w, "Task not found", http.StatusNotFound)
	}
}

// handleClearTasks 清空所有任务
func handleClearTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	debugLog("🌐 [HTTP] %s %s | 清空所有任务", r.Method, r.URL.Path)

	// 清空所有任务
	count := globalSessionManager.Taskmng.ClearAllTasks()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("已清空 %d 个任务", count),
		"count":   count,
	})
}

// handleTaskStatus 返回任务状态列表
func handleTaskStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 从 TaskManager 获取所有任务状态
	tasks := globalSessionManager.Taskmng.GetAllTasks()
	json.NewEncoder(w).Encode(tasks)
}

// handleGetFormat 获取当前格式化字符串
func handleGetFormat(w http.ResponseWriter, r *http.Request) {
	debugLog("🌐 [HTTP] %s %s | 获取格式化字符串", r.Method, r.URL.Path)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"format": Format,
	})
}

// handleSetFormat 设置格式化字符串
func handleSetFormat(w http.ResponseWriter, r *http.Request) {
	debugLog("🌐 [HTTP] %s %s | 设置格式化字符串", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		debugLog("❌ [HTTP] 方法不允许 | %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Format string `json:"format"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		debugLog("❌ [HTTP] 请求体解析失败 | %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Format == "" {
		debugLog("❌ [HTTP] 格式化字符串不能为空")
		http.Error(w, "Format cannot be empty", http.StatusBadRequest)
		return
	}

	// 更新全局格式化字符串
	Format = req.Format
	debugLog("✅ [HTTP] 格式化字符串已更新 | 新值: %s", Format)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Format updated",
		"format":  Format,
	})
}
