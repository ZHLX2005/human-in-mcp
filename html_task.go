package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// TaskRequest 任务请求结构
type TaskRequest struct {
	ConversationID string `json:"conversationId"`
	CustomInput    string `json:"customInput"`
	Continue       bool   `json:"continue"`
	SelectedIndex  *int   `json:"selectedIndex"` // 可选，从AI选项中选择
}

// 启动HTTP服务器
func StartTaskServer() {
	// API路由
	http.HandleFunc("/", serveHomePage)
	http.HandleFunc("/api/tasks", handleTasks)
	http.HandleFunc("/api/tasks/list", handleListTasks)
	http.HandleFunc("/api/render-tasks", handleRenderTasks)
	http.HandleFunc("/api/render-tasks/select", handleSelectRenderTask)

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
            grid-template-columns: 1fr 1fr;
            gap: 20px;
        }
        .panel {
            background: white;
            border-radius: 12px;
            box-shadow: 0 2px 12px rgba(0,0,0,0.08);
            overflow: hidden;
        }
        .header {
            background: #ffffff;
            color: #333;
            padding: 20px;
            text-align: center;
            border-bottom: 1px solid #e8e8e8;
        }
        .header h2 {
            font-size: 18px;
            font-weight: 600;
            color: #1a1a1a;
        }
        .header p {
            font-size: 12px;
            color: #999;
            margin-top: 5px;
        }
        .content { padding: 20px; }
        .tabs {
            display: flex;
            border-bottom: 1px solid #e8e8e8;
            margin-bottom: 20px;
        }
        .tab {
            flex: 1;
            padding: 12px;
            text-align: center;
            cursor: pointer;
            font-size: 14px;
            color: #666;
            border-bottom: 2px solid transparent;
            transition: all 0.2s;
        }
        .tab:hover {
            background: #fafafa;
        }
        .tab.active {
            color: #333;
            border-bottom-color: #333;
            font-weight: 600;
        }
        .tab-content { display: none; }
        .tab-content.active { display: block; }
        .form-group { margin-bottom: 15px; }
        .form-group label {
            display: block;
            margin-bottom: 6px;
            font-weight: 500;
            color: #333;
            font-size: 13px;
        }
        .form-group input, .form-group textarea, .form-group select {
            width: 100%;
            padding: 10px;
            border: 1px solid #e0e0e0;
            border-radius: 6px;
            font-size: 13px;
            transition: all 0.2s;
            background: #fafafa;
        }
        .form-group input:focus, .form-group textarea:focus, .form-group select:focus {
            outline: none;
            border-color: #999;
            background: white;
        }
        .form-group textarea {
            min-height: 80px;
            resize: vertical;
        }
        .btn {
            width: 100%;
            padding: 10px 20px;
            border: 1px solid #e0e0e0;
            border-radius: 6px;
            font-size: 13px;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.2s;
            background: white;
            color: #333;
            margin-bottom: 10px;
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
        .btn-group { display: flex; gap: 8px; }
        .btn-group .btn { margin-bottom: 0; }
        .task-list h3, .render-list h3 {
            font-size: 14px;
            margin-bottom: 12px;
            color: #333;
            font-weight: 600;
        }
        .task-item, .render-item {
            background: #fafafa;
            padding: 12px;
            border-radius: 6px;
            margin-bottom: 8px;
            border-left: 3px solid #999;
        }
        .task-item:hover, .render-item:hover {
            background: #f0f0f0;
        }
        .task-item .task-id, .render-item .render-id {
            font-size: 10px;
            color: #999;
            margin-bottom: 4px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        .task-item .task-content, .render-item .summary {
            font-size: 13px;
            color: #333;
            margin-bottom: 6px;
            line-height: 1.5;
        }
        .task-item .task-meta, .render-item .render-meta {
            font-size: 11px;
            color: #888;
        }
        .render-item .options {
            margin-top: 8px;
            display: flex;
            flex-wrap: wrap;
            gap: 6px;
        }
        .option-btn {
            padding: 4px 10px;
            font-size: 11px;
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
        .empty-state {
            text-align: center;
            padding: 30px;
            color: #aaa;
            font-size: 13px;
        }
        .message {
            padding: 10px;
            border-radius: 6px;
            margin-bottom: 12px;
            display: none;
            font-size: 13px;
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
        .badge {
            display: inline-block;
            padding: 2px 8px;
            font-size: 10px;
            border-radius: 4px;
            background: #f0f0f0;
            color: #666;
            margin-left: 8px;
        }
        @media (max-width: 768px) {
            .container { grid-template-columns: 1fr; }
        }
    </style>
</head>
<body>
    <div class="container">
        <!-- 左侧：手动添加任务 -->
        <div class="panel">
            <div class="header">
                <h2>📝 手动添加任务</h2>
                <p>直接添加任务到队列</p>
            </div>
            <div class="content">
                <div id="manualMessage" class="message"></div>

                <form id="manualTaskForm">
                    <div class="form-group">
                        <label for="manualConversationId">对话ID</label>
                        <input type="text" id="manualConversationId" placeholder="例如: session-123" required>
                    </div>

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

                <div class="task-list">
                    <h3>任务队列 <span id="taskCount" class="badge">0</span></h3>
                    <div id="taskList">
                        <div class="empty-state">暂无任务</div>
                    </div>
                </div>
            </div>
        </div>

        <!-- 右侧：AI渲染任务 -->
        <div class="panel">
            <div class="header">
                <h2>🤖 AI 渲染任务</h2>
                <p>处理AI发送的交互请求</p>
            </div>
            <div class="content">
                <div id="renderMessage" class="message"></div>

                <div class="render-list">
                    <h3>待处理任务 <span id="renderCount" class="badge">0</span></h3>
                    <div id="renderList">
                        <div class="empty-state">暂无AI任务</div>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <script>
        // 手动任务表单
        document.getElementById('manualTaskForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const messageEl = document.getElementById('manualMessage');

            const task = {
                conversationId: document.getElementById('manualConversationId').value,
                customInput: document.getElementById('manualCustomInput').value,
                continue: document.getElementById('manualContinueTask').value === 'true'
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
                    loadTasks();
                } else {
                    showMessage('manualMessage', '添加失败：' + (await response.text()), 'error');
                }
            } catch (error) {
                showMessage('manualMessage', '网络错误：' + error.message, 'error');
            }
        });

        // 加入手动任务列表
        async function loadTasks() {
            try {
                const response = await fetch('/api/tasks/list');
                const tasks = await response.json();

                const taskList = document.getElementById('taskList');
                document.getElementById('taskCount').textContent = tasks.length;

                if (tasks.length === 0) {
                    taskList.innerHTML = '<div class="empty-state">暂无任务</div>';
                } else {
                    taskList.innerHTML = tasks.map((task, index) => {
                        return '<div class="task-item">' +
                            '<div class="task-id">#' + (index + 1) + ' | ' + task.conversationId + '</div>' +
                            '<div class="task-content">' + escapeHtml(task.customInput) + '</div>' +
                            '<div class="task-meta">类型: ' + (task.continue ? '继续任务' : '结束对话') + '</div>' +
                            '</div>';
                    }).join('');
                }
            } catch (error) {
                console.error('加载任务列表失败:', error);
            }
        }

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
                                optionsHtml += '<button class="option-btn" onclick="selectOption(\'' + task.conversationId + '\', ' + i + ')">[' + (i + 1) + '] ' + escapeHtml(opt.substring(0, 20)) + '</button>';
                            });
                            optionsHtml += '<button class="option-btn" onclick="showCustomInput(\'' + task.conversationId + '\')">[自定义]</button>';
                            optionsHtml += '<button class="option-btn" onclick="endChat(\'' + task.conversationId + '\')">[结束]</button>';
                            optionsHtml += '</div>';
                        }

                        return '<div class="render-item">' +
                            '<div class="render-id">' + task.conversationId + '</div>' +
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

        // 选择AI选项
        async function selectOption(conversationId, index) {
            const task = {
                conversationId: conversationId,
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
                    showMessage('renderMessage', '已选择选项！', 'success');
                    loadRenderTasks();
                    loadTasks();
                } else {
                    showMessage('renderMessage', '选择失败：' + (await response.text()), 'error');
                }
            } catch (error) {
                showMessage('renderMessage', '网络错误：' + error.message, 'error');
            }
        }

        // 自定义输入
        function showCustomInput(conversationId) {
            const customInput = prompt('请输入您的指示:');
            if (customInput === null || customInput.trim() === '') return;

            const task = {
                conversationId: conversationId,
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
                    showMessage('renderMessage', '已提交自定义输入！', 'success');
                    loadRenderTasks();
                    loadTasks();
                } else {
                    showMessage('renderMessage', '提交失败', 'error');
                }
            });
        }

        // 结束对话
        async function endChat(conversationId) {
            const task = {
                conversationId: conversationId,
                continue: false,
                customInput: '用户选择结束对话'
            };

            try {
                const response = await fetch('/api/render-tasks/select', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(task)
                });

                if (response.ok) {
                    showMessage('renderMessage', '已结束对话！', 'success');
                    loadRenderTasks();
                    loadTasks();
                } else {
                    showMessage('renderMessage', '操作失败：' + (await response.text()), 'error');
                }
            } catch (error) {
                showMessage('renderMessage', '网络错误：' + error.message, 'error');
            }
        }

        function showMessage(elementId, text, type) {
            const message = document.getElementById(elementId);
            message.textContent = text;
            message.className = 'message ' + type;
            message.style.display = 'block';
            setTimeout(() => { message.style.display = 'none'; }, 3000);
        }

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        // 页面加载时获取数据
        loadTasks();
        loadRenderTasks();
        // 每3秒自动刷新
        setInterval(() => {
            loadTasks();
            loadRenderTasks();
        }, 3000);
    </script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(tmpl))
}

// handleTasks 处理手动任务添加请求
func handleTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var task TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 验证必填字段
	if task.ConversationID == "" || task.CustomInput == "" {
		http.Error(w, "conversationId and customInput are required", http.StatusBadRequest)
		return
	}

	// 创建响应并添加到队列
	response := UserChoiceResponse{
		ConversationID: task.ConversationID,
		CustomInput:    task.CustomInput,
		Continue:       task.Continue,
		SelectedIndex:  -1,
	}

	globalSessionManager.PushResponse(response)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Task added to queue",
	})
}

// handleListTasks 返回当前任务列表
func handleListTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(globalSessionManager.GetResponses())
}

// handleRenderTasks 返回AI渲染任务列表
func handleRenderTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(globalSessionManager.GetRenderTasks())
}

// handleSelectRenderTask 处理从AI渲染任务中选择选项
func handleSelectRenderTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ConversationID string `json:"conversationId"`
		SelectedIndex  *int   `json:"selectedIndex"`
		CustomInput    string `json:"customInput"`
		Continue       bool   `json:"continue"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 获取渲染任务
	renderTasks := globalSessionManager.GetRenderTasks()
	var targetTask *RenderTask
	var taskIndex int

	for i, task := range renderTasks {
		if task.ConversationID == req.ConversationID {
			targetTask = &task
			taskIndex = i
			break
		}
	}

	if targetTask == nil {
		http.Error(w, "Render task not found", http.StatusNotFound)
		return
	}

	// 创建响应
	response := UserChoiceResponse{
		ConversationID: req.ConversationID,
		Continue:       req.Continue,
		SelectedIndex:  -1,
	}

	if req.SelectedIndex != nil && *req.SelectedIndex >= 0 && *req.SelectedIndex < len(targetTask.NextOptions) {
		// 从AI选项中选择
		response.SelectedIndex = *req.SelectedIndex
		response.CustomInput = targetTask.NextOptions[*req.SelectedIndex]
	} else if req.CustomInput != "" {
		// 自定义输入
		response.CustomInput = req.CustomInput
		response.SelectedIndex = -1
	} else {
		// 结束对话
		response.CustomInput = "用户选择结束对话"
	}

	// 发送到Out通道
	globalSessionManager.PushResponse(response)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Response sent",
		"index":   strconv.Itoa(taskIndex),
	})
}
