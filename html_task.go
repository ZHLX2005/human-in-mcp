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
                <h2>📝 手动添加任务</h2>
                <p>直接添加任务到队列</p>
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

                <div class="list-header">
                    <span>任务队列</span>
                    <span id="taskCount" class="badge">0</span>
                </div>
                <div id="taskList">
                    <div class="empty-state">暂无任务</div>
                </div>
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
        // 手动任务表单
        document.getElementById('manualTaskForm').addEventListener('submit', async (e) => {
            e.preventDefault();

            const task = {
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
                    loadTaskStatus();
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
                            '<div class="task-content">' + escapeHtml(task.customInput) + '</div>' +
                            '<div class="task-meta">类型: ' + (task.continue ? '继续' : '结束') + '</div>' +
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
                                optionsHtml += '<button class="option-btn" onclick="selectOption(' + i + ', \'' + escapeHtml(opt).replace(/'/g, "\\'") + '\')">[' + (i + 1) + '] ' + escapeHtml(opt.substring(0, 15)) + '</button>';
                            });
                            optionsHtml += '<button class="option-btn" onclick="showCustomInput()">自定义</button>';
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

                        return '<div class="status-item ' + task.status + '">' +
                            '<div class="task-id">ID: ' + escapeHtml(task.taskId) + '</div>' +
                            statusBadge +
                            '<div class="task-req">' + escapeHtml(task.req) + '</div>' +
                            respHtml +
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
        loadTasks();
        loadRenderTasks();
        loadTaskStatus();
        // 每2秒自动刷新
        setInterval(() => {
            loadTasks();
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
	if task.CustomInput == "" {
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Response sent",
	})
}

// handleTaskStatus 返回任务状态列表
func handleTaskStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 从 TaskManager 获取所有任务状态
	tasks := globalSessionManager.Taskmng.GetAllTasks()
	json.NewEncoder(w).Encode(tasks)
}
