claude mcp add --transport sse weather-auth http://localhost:8092/sse -e Authorization="Bearer my-secret-token"

claude mcp add --transport sse human-in-mcp http://localhost:8093/sse

添加相关的mcp

claude

添加全局的

claude mcp add --transport sse --scope user human-in-mcp http://localhost:8093/sse

请使用mcp当中 human-in-mcp 的  human_interaction工具

3

请使用mcp当中 human-in-mcp 的  human_interaction工具

claude --dangerously-skip-permissions 请使用mcp当中human-in-mcp的human_interaction工具
