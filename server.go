package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const AUTH_TOKEN = "my-secret-token"

// Weather 工具定义
type WeatherResponse struct {
	City        string        `json:"city"`
	Temperature string        `json:"temperature"`
	Condition   string        `json:"condition"`
	Forecast    []ForecastDay `json:"forecast,omitempty"`
}

type ForecastDay struct {
	Date      string `json:"date"`
	High      string `json:"high"`
	Low       string `json:"low"`
	Condition string `json:"condition"`
}

func WeatherTool() mcp.Tool {
	return mcp.NewTool(
		"get_weather",
		mcp.WithDescription("获取指定城市天气信息"),
		mcp.WithString("city", mcp.Required(), mcp.Description("城市名称")),
		mcp.WithString("extensions", mcp.Required(),
			mcp.Enum("base", "all"),
			mcp.DefaultString("base"),
			mcp.Description("返回数据类型 base=实况，all=预报"),
		),
	)
}

type KnowledgeDoc struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

func weatherHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	city, _ := req.RequireString("city")
	ext, _ := req.RequireString("extensions")

	fmt.Printf("协议Call %s", req)
	resp := WeatherResponse{
		City:        city,
		Temperature: "25°C",
		Condition:   "Sunny",
	}
	if ext == "all" {
		resp.Forecast = []ForecastDay{
			{Date: time.Now().Add(24 * time.Hour).Format("2006-01-02"), High: "28°C", Low: "19°C", Condition: "Cloudy"},
			{Date: time.Now().Add(48 * time.Hour).Format("2006-01-02"), High: "26°C", Low: "18°C", Condition: "Rain"},
		}
	}

	data, _ := json.Marshal(resp)
	return mcp.NewToolResultText(string(data)), nil
}

func main() {
	mcpServer := server.NewMCPServer("weather-auth", "v1.0.0")
	mcpServer.AddTool(WeatherTool(), weatherHandler)
	// mcpServer.AddTool(KnowledgeBaseTool(), knowledgeHandler)

	sseServer := server.NewSSEServer(mcpServer)

	// 使用 ServeMux 来正确路由请求
	mux := http.NewServeMux()

	// 认证中间件
	authMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("Authorization")
			if token != "Bearer "+AUTH_TOKEN {
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprintln(w, "Unauthorized: invalid token")
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	// 将 SSE 服务器的所有路由都注册到 mux，并应用认证
	mux.Handle("/", authMiddleware(sseServer))

	fmt.Println("✅ MCP SSE Server with Auth running on http://localhost:8092")
	if err := http.ListenAndServe("localhost:8092", mux); err != nil {
		panic(err)
	}
}
