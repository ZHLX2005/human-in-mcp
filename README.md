# Human-In-MCP

让 AI 在完成任务后通过 MCP 向用户展示信息并获取下一步指示，实现无限对话循环。

## 功能特点

- **任务总结展示**: AI 完成任务后向用户展示工作总结
- **困难反馈**: 显示 AI 遇到的问题或需要的帮助
- **选项交互**: 用户可以从预设选项中选择或输入自定义指令
- **多轮对话**: 支持通过 ConversationID 跟踪对话状态
- **无限循环**: AI 可以持续与用户交互，直到用户选择结束

## 安装运行

```bash
go run human_in_mcp_server.go
```

服务器将在 `http://localhost:8093` 启动。

## MCP 配置

在 Claude Desktop 或其他 MCP 客户端配置：

```json
{
  "mcpServers": {
    "human-in-mcp": {
      "command": "go",
      "args": ["run", "d:/code/a_go/proj/human_in_mcp/human_in_mcp_server.go"],
      "env": {}
    }
  }
}
```

## 工具使用

### human_interaction

AI 调用此工具向用户展示信息并获取下一步指示。

**参数:**

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `summary` | string | 是 | 完成任务的简单总结 |
| `difficulties` | string | 是 | 遇到的困难、需要的帮助或其他重要信息 |
| `conversationId` | string | 是 | 对话ID，用于跟踪多轮对话（建议使用时间戳或UUID） |
| `nextOptions` | string | 是 | 可选项的 JSON 数组字符串，如 `["继续", "修改", "结束"]` |

**返回:**

```json
{
  "conversationId": "string",
  "selectedIndex": 0,    // -1 表示自定义输入
  "customInput": "string",
  "continue": true
}
```

## AI 使用示例

当 AI 完成任务后，可以这样调用：

```
我已完成代码重构工作。
调用 human_interaction 工具向用户展示:
- summary: "重构了用户认证模块，将代码从500行减少到300行，提高了可读性"
- difficulties: "部分遗留代码逻辑复杂，建议后续逐步重构"
- conversationId: "task-001-1738250000"
- nextOptions: ["继续优化其他模块", "添加单元测试", "提交代码", "查看具体改动"]
```

## 用户交互示例

```
======================================================================
🤖 AI 任务完成报告 [对话ID: task-001-1738250000]
======================================================================

📋 任务总结:
重构了用户认证模块，将代码从500行减少到300行，提高了可读性

⚠️  遇到的问题/需要的帮助:
部分遗留代码逻辑复杂，建议后续逐步重构

🔄 接下来的可选项:
  [1] 继续优化其他模块
  [2] 添加单元测试
  [3] 提交代码
  [4] 查看具体改动
  [0] 自定义输入
  [q] 结束对话

----------------------------------------------------------------------

请选择操作 (输入数字或命令): 2

✅ 已记录您的选择，将继续执行...
======================================================================
```

## 工作流程

```
┌─────────────────────────────────────────────────────────────────┐
│                        AI Agent                                  │
│  1. 执行任务                                                     │
│  2. 调用 human_interaction 工具                                  │
│     ┌─────────────────────────────────────┐                    │
│     │ 传入: summary, difficulties,         │                    │
│     │       conversationId, nextOptions    │                    │
│     └─────────────────────────────────────┘                    │
│                              ↓                                  │
│  3. 等待用户选择                                                 │
│                              ↑                                  │
│     ┌─────────────────────────────────────┐                    │
│     │ 返回: selectedIndex, customInput,   │                    │
│     │       continue                       │                    │
│     └─────────────────────────────────────┘                    │
│  4. 根据 continue 决定:                                         │
│     - true: 继续执行下一步任务 → 回到步骤 1                      │
│     - false: 结束对话                                           │
└─────────────────────────────────────────────────────────────────┘
                              ↕
                    ┌─────────────────┐
                    │   用户控制台     │
                    │  查看任务报告    │
                    │  选择下一步操作  │
                    └─────────────────┘
```

## 与 AI Agent 集成示例

```python
# AI Agent 伪代码示例
def execute_with_human_loop(agent):
    conversation_id = str(int(time.time()))

    while True:
        # 1. AI 执行任务
        result = agent.execute()

        # 2. 向用户展示并获取下一步指示
        response = mcp.call("human_interaction", {
            "summary": result.summary,
            "difficulties": result.difficulties,
            "conversationId": conversation_id,
            "nextOptions": json.dumps(result.suggestions)
        })

        # 3. 检查是否继续
        if not response["continue"]:
            break

        # 4. 根据用户选择继续
        if response["selectedIndex"] == -1:
            agent.set_custom_task(response["customInput"])
        else:
            agent.select_option(response["selectedIndex"])
```

## 注意事项

1. **conversationId** 应该在同一个对话循环中保持一致，用于跟踪对话状态
2. **nextOptions** 必须是有效的 JSON 数组字符串格式
3. 用户输入 `q` 可随时结束对话
4. 选择 `0` 可输入自定义指令
