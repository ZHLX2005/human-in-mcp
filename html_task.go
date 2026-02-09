package main

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	http.HandleFunc("/api/render-tasks", handleRenderTasks)
	http.HandleFunc("/api/render-tasks/select", handleSelectRenderTask)
	http.HandleFunc("/api/render-tasks/abandon", handleAbandonRenderTask) // 遗弃AI渲染任务

	fmt.Println("📝 任务管理页面: http://localhost:8094")
	go http.ListenAndServe(":8094", nil)
}

// serveHomePage 提供主页HTML
func serveHomePage(w http.ResponseWriter, r *http.Request) {
	tmpl := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>任务队列管理</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #f5f5f5;
            min-height: 100vh;
            padding: 20px;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            display: grid;
            grid-template-columns: 1fr 1fr 1fr;
            gap: 20px;
        }
        .panel {
            background: white;
            border-radius: 12px;
            box-shadow: 0 2px 12px rgba(0,0,0,0.08);
            overflow: hidden;
            display: flex;
            flex-direction: column;
            max-height: calc(100vh - 40px);
        }
        .header {
            background: #ffffff;
            color: #333;
            padding: 16px 20px;
            text-align: center;
            border-bottom: 1px solid #e8e8e8;
            flex-shrink: 0;
        }
        .header h2 {
            font-size: 16px;
            font-weight: 600;
            color: #1a1a1a;
        }
        .header p {
            font-size: 11px;
            color: #999;
            margin-top: 4px;
        }
        .content {
            padding: 16px;
            overflow-y: auto;
            flex: 1;
        }
        .form-group { margin-bottom: 12px; }
        .form-group label {
            display: block;
            margin-bottom: 5px;
            font-weight: 500;
            color: #333;
            font-size: 12px;
        }
        .form-group input, .form-group textarea, .form-group select {
            width: 100%;
            padding: 8px 10px;
            border: 1px solid #e0e0e0;
            border-radius: 6px;
            font-size: 12px;
            transition: all 0.2s;
            background: #fafafa;
        }
        .form-group input:focus, .form-group textarea:focus, .form-group select:focus {
            outline: none;
            border-color: #999;
            background: white;
        }
        .form-group textarea {
            min-height: 60px;
            resize: vertical;
        }
        .btn {
            width: 100%;
            padding: 8px 16px;
            border: 1px solid #e0e0e0;
            border-radius: 6px;
            font-size: 12px;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.2s;
            background: white;
            color: #333;
            margin-bottom: 8px;
        }
        .btn:hover {
            background: #f5f5f5;
            border-color: #ccc;
        }
        .btn-primary {
            background: #333;
            color: white;
            border-color: #333;
        }
        .btn-primary:hover {
            background: #555;
            border-color: #555;
        }
        .list-header {
            font-size: 13px;
            margin-bottom: 10px;
            color: #333;
            font-weight: 600;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .badge {
            padding: 2px 8px;
            font-size: 10px;
            border-radius: 4px;
            background: #f0f0f0;
            color: #666;
        }
        .task-item, .render-item {
            background: #fafafa;
            padding: 10px;
            border-radius: 6px;
            margin-bottom: 6px;
            border-left: 3px solid #999;
            font-size: 12px;
        }
        .task-item:hover, .render-item:hover {
            background: #f0f0f0;
        }
        .task-content, .render-item .summary {
            color: #333;
            margin-bottom: 4px;
            line-height: 1.4;
        }
        .task-meta, .render-meta {
            font-size: 10px;
            color: #888;
        }
        .render-item .options {
            margin-top: 8px;
            display: flex;
            flex-wrap: wrap;
            gap: 4px;
        }
        .option-btn {
            padding: 3px 8px;
            font-size: 10px;
            background: white;
            border: 1px solid #e0e0e0;
            border-radius: 4px;
            cursor: pointer;
            transition: all 0.2s;
        }
        .option-btn:hover {
            background: #333;
            color: white;
            border-color: #333;
        }
        .processed-item {
            background: #e8f5e9;
            padding: 10px;
            border-radius: 6px;
            margin-bottom: 6px;
            border-left: 3px solid #4caf50;
            font-size: 12px;
            opacity: 0.9;
        }
        .processed-item .summary {
            color: #333;
            margin-bottom: 4px;
            line-height: 1.4;
        }
        .processed-item .response {
            color: #2e7d32;
            font-size: 11px;
            padding: 4px 8px;
            background: white;
            border-radius: 4px;
            margin-top: 4px;
        }
        .processed-item .timestamp {
            font-size: 9px;
            color: #888;
            margin-top: 4px;
        }
        .status-item {
            background: #f8f9fa;
            padding: 10px;
            border-radius: 6px;
            margin-bottom: 6px;
            border-left: 3px solid #6c757d;
            font-size: 12px;
        }
        .status-item.pending {
            border-left-color: #ffc107;
            background: #fff8e1;
        }
        .status-item.processing {
            border-left-color: #2196f3;
            background: #e3f2fd;
        }
        .status-item.completed {
            border-left-color: #4caf50;
            background: #e8f5e9;
        }
        .status-item .task-id {
            font-size: 9px;
            color: #888;
            margin-bottom: 4px;
        }
        .status-item .task-req {
            color: #333;
            margin-bottom: 4px;
            line-height: 1.4;
        }
        .status-item .task-resp {
            color: #2e7d32;
            font-size: 11px;
            padding: 4px 8px;
            background: white;
            border-radius: 4px;
            margin-top: 4px;
        }
        .status-badge {
            display: inline-block;
            padding: 2px 8px;
            font-size: 9px;
            border-radius: 4px;
            font-weight: 500;
            margin-bottom: 4px;
        }
        .status-badge.pending {
            background: #ffc107;
            color: #333;
        }
        .status-badge.processing {
            background: #2196f3;
            color: white;
        }
        .status-badge.completed {
            background: #4caf50;
            color: white;
        }
        .empty-state {
            text-align: center;
            padding: 30px 20px;
            color: #aaa;
            font-size: 12px;
        }
        .message {
            padding: 8px;
            border-radius: 6px;
            margin-bottom: 10px;
            display: none;
            font-size: 12px;
        }
        .message.success {
            background: #f0f5f0;
            color: #2d502d;
            border: 1px solid #c8e6c9;
        }
        .message.error {
            background: #fef0f0;
            color: #c62828;
            border: 1px solid #ffcdd2;
        }
        @media (max-width: 1200px) {
            .container { grid-template-columns: 1fr; }
            .panel { max-height: none; }
        }
    </style>
</head>
<body>
    <div class="container">
        <!-- 左侧：手动添加任务 -->
        <div class="panel">
            <div class="header">
                <h2>📝 添加待处理任务</h2>
                <p>创建新的待处理任务</p>
            </div>
            <div class="content">
                <div id="manualMessage" class="message"></div>

                <form id="manualTaskForm">
                    <div class="form-group">
                        <label for="manualCustomInput">任务内容</label>
                        <textarea id="manualCustomInput" placeholder="请输入任务描述..." required></textarea>
                    </div>

                    <div class="form-group">
                        <label for="manualContinueTask">任务类型</label>
                        <select id="manualContinueTask">
                            <option value="true">继续任务</option>
                            <option value="false">结束对话</option>
                        </select>
                    </div>

                    <button type="submit" class="btn btn-primary">添加任务</button>
                </form>
            </div>
        </div>

        <!-- 中间：AI渲染任务 -->
        <div class="panel">
            <div class="header">
                <h2>🤖 AI 渲染任务</h2>
                <p>处理AI发送的交互请求</p>
            </div>
            <div class="content">
                <div id="renderMessage" class="message"></div>

                <div class="list-header">
                    <span>待处理任务</span>
                    <span id="renderCount" class="badge">0</span>
                </div>
                <div id="renderList">
                    <div class="empty-state">暂无AI任务</div>
                </div>
            </div>
        </div>

        <!-- 右侧：任务状态 -->
        <div class="panel">
            <div class="header">
                <h2>📊 任务状态</h2>
                <p>实时追踪任务进度</p>
            </div>
            <div class="content">
                <div class="list-header">
                    <span>全部任务</span>
                    <span id="statusCount" class="badge">0</span>
                </div>
                <div id="statusList">
                    <div class="empty-state">暂无任务状态</div>
                </div>
            </div>
        </div>
    </div>

    <script>
        // 监听任务类型变化
        document.getElementById('manualContinueTask').addEventListener('change', (e) => {
            const customInput = document.getElementById('manualCustomInput');
            if (e.target.value === 'false') {
                // 结束对话，自动填充文本并禁用输入框
                customInput.value = '结束任务';
                customInput.disabled = true;
                customInput.required = false;
            } else {
                // 继续任务，启用输入框
                customInput.disabled = false;
                customInput.required = true;
                customInput.value = '';
            }
        });

        // 手动任务表单
        document.getElementById('manualTaskForm').addEventListener('submit', async (e) => {
            e.preventDefault();

            const isContinue = document.getElementById('manualContinueTask').value === 'true';
            const task = {
                customInput: isContinue ? document.getElementById('manualCustomInput').value : '结束任务',
                continue: isContinue
            };

            try {
                const response = await fetch('/api/tasks', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(task)
                });

                if (response.ok) {
                    showMessage('manualMessage', '任务添加成功！', 'success');
                    document.getElementById('manualTaskForm').reset();
                    // 重置后重新启用输入框
                    document.getElementById('manualCustomInput').disabled = false;
                    document.getElementById('manualCustomInput').required = true;
                    loadTaskStatus();
                } else {
                    showMessage('manualMessage', '添加失败：' + (await response.text()), 'error');
                }
            } catch (error) {
                showMessage('manualMessage', '网络错误：' + error.message, 'error');
            }
        });

        // 加载AI渲染任务列表
        async function loadRenderTasks() {
            try {
                const response = await fetch('/api/render-tasks');
                const tasks = await response.json();

                const renderList = document.getElementById('renderList');
                document.getElementById('renderCount').textContent = tasks.length;

                if (tasks.length === 0) {
                    renderList.innerHTML = '<div class="empty-state">暂无AI任务</div>';
                } else {
                    renderList.innerHTML = tasks.map((task, index) => {
                        let optionsHtml = '';
                        if (task.nextOptions && task.nextOptions.length > 0) {
                            optionsHtml = '<div class="options">';
                            task.nextOptions.forEach((opt, i) => {
                                optionsHtml += '<button class="option-btn" onclick="selectOption(' + i + ', \'' + escapeHtml(opt).replace(/'/g, "\\'") + '\')">[' + (i + 1) + '] ' + escapeHtml(opt.substring(0, 15)) + '</button>';
                            });
                            optionsHtml += '<button class="option-btn" onclick="showCustomInput()">自定义</button>';
                            optionsHtml += '<button class="option-btn" onclick="abandonTask()">遗弃</button>';
                            optionsHtml += '<button class="option-btn" onclick="endChat()">结束</button>';
                            optionsHtml += '</div>';
                        }

                        return '<div class="render-item">' +
                            '<div class="summary">' + escapeHtml(task.summary) + '</div>' +
                            (task.difficulties && task.difficulties !== '无' ? '<div class="render-meta">⚠️ ' + escapeHtml(task.difficulties) + '</div>' : '') +
                            optionsHtml +
                            '</div>';
                    }).join('');
                }
            } catch (error) {
                console.error('加载渲染任务失败:', error);
            }
        }

        // 加载任务状态
        async function loadTaskStatus() {
            try {
                const response = await fetch('/api/tasks/status');
                const tasks = await response.json();

                const statusList = document.getElementById('statusList');
                document.getElementById('statusCount').textContent = tasks.length;

                if (tasks.length === 0) {
                    statusList.innerHTML = '<div class="empty-state">暂无任务状态</div>';
                } else {
                    statusList.innerHTML = tasks.map(task => {
                        let statusBadge = '';
                        switch(task.status) {
                            case 'pending':
                                statusBadge = '<span class="status-badge pending">等待中</span>';
                                break;
                            case 'processing':
                                statusBadge = '<span class="status-badge processing">处理中</span>';
                                break;
                            case 'completed':
                                statusBadge = '<span class="status-badge completed">已完成</span>';
                                break;
                            default:
                                statusBadge = '<span class="status-badge">' + task.status + '</span>';
                        }

                        let respHtml = '';
                        if (task.resp && task.resp !== '') {
                            respHtml = '<div class="task-resp">↳ ' + escapeHtml(task.resp) + '</div>';
                        }

                        // 为pending状态的任务添加删除按钮
                        let deleteBtn = '';
                        if (task.status === 'pending') {
                            deleteBtn = '<button class="option-btn" onclick="deleteTask(\'' + escapeHtml(task.taskId) + '\')" style="margin-top: 4px; background: #f44336; color: white; border-color: #f44336;">删除</button>';
                        }

                        return '<div class="status-item ' + task.status + '">' +
                            '<div class="task-id">ID: ' + escapeHtml(task.taskId) + '</div>' +
                            statusBadge +
                            '<div class="task-req">' + escapeHtml(task.req) + '</div>' +
                            respHtml +
                            deleteBtn +
                            '</div>';
                    }).join('');
                }
            } catch (error) {
                console.error('加载任务状态失败:', error);
            }
        }

        // 选择AI选项
        async function selectOption(index, optionText) {
            const task = {
                selectedIndex: index,
                continue: true,
                customInput: ''
            };

            try {
                const response = await fetch('/api/render-tasks/select', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(task)
                });

                if (response.ok) {
                    showMessage('renderMessage', '已选择: ' + optionText, 'success');
                    loadRenderTasks();
                    loadTaskStatus();
                } else {
                    showMessage('renderMessage', '选择失败', 'error');
                }
            } catch (error) {
                showMessage('renderMessage', '网络错误', 'error');
            }
        }

        // 自定义输入
        function showCustomInput() {
            const customInput = prompt('请输入您的指示:');
            if (customInput === null || customInput.trim() === '') return;

            const task = {
                selectedIndex: -1,
                continue: true,
                customInput: customInput
            };

            fetch('/api/render-tasks/select', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(task)
            }).then(response => {
                if (response.ok) {
                    showMessage('renderMessage', '已提交', 'success');
                    loadRenderTasks();
                    loadTaskStatus();
                }
            });
        }

        // 结束对话
        async function endChat() {
            const task = {
                continue: false,
                customInput: '结束对话'
            };

            try {
                const response = await fetch('/api/render-tasks/select', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(task)
                });

                if (response.ok) {
                    showMessage('renderMessage', '已结束对话', 'success');
                    loadRenderTasks();
                    loadTaskStatus();
                }
            } catch (error) {
                showMessage('renderMessage', '操作失败', 'error');
            }
        }

        // 遗弃任务
        async function abandonTask() {
            if (!confirm('确定要遗弃这个任务吗？遗弃后任务将被移除。')) {
                return;
            }

            try {
                const response = await fetch('/api/render-tasks/abandon', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' }
                });

                if (response.ok) {
                    showMessage('renderMessage', '任务已遗弃', 'success');
                    loadRenderTasks();
                    loadTaskStatus();
                } else {
                    showMessage('renderMessage', '遗弃失败', 'error');
                }
            } catch (error) {
                showMessage('renderMessage', '网络错误', 'error');
            }
        }

        // 删除任务
        async function deleteTask(taskId) {
            if (!confirm('确定要删除这个任务吗？')) {
                return;
            }

            try {
                const response = await fetch('/api/tasks/delete', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ taskId: taskId })
                });

                if (response.ok) {
                    loadTaskStatus();
                } else {
                    alert('删除失败');
                }
            } catch (error) {
                alert('网络错误');
            }
        }

        function showMessage(elementId, text, type) {
            const message = document.getElementById(elementId);
            message.textContent = text;
            message.className = 'message ' + type;
            message.style.display = 'block';
            setTimeout(() => { message.style.display = 'none'; }, 2000);
        }

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        // 页面加载时获取数据
        loadRenderTasks();
        loadTaskStatus();
        // 每2秒自动刷新
        setInterval(() => {
            loadRenderTasks();
            loadTaskStatus();
        }, 2000);
    </script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(tmpl))
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

// handleTaskStatus 返回任务状态列表
func handleTaskStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 从 TaskManager 获取所有任务状态
	tasks := globalSessionManager.Taskmng.GetAllTasks()
	json.NewEncoder(w).Encode(tasks)
}
