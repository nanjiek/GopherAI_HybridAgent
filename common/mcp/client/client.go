package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// comment cleaned
// comment cleaned
	c *client.Client
}

// comment cleaned
func NewMCPClient(httpURL string) (*MCPClient, error) {
	fmt.Println("姝ｅ湪鍒濆鍖朒TTP瀹㈡埛绔?..")
	// comment cleaned
	httpTransport, err := transport.NewStreamableHTTP(httpURL)
	if err != nil {
		return nil, fmt.Errorf("鍒涘缓HTTP浼犺緭澶辫触: %w", err)
	}

	// comment cleaned

	return &MCPClient{c: c}, nil
}

// comment cleaned
func (m *MCPClient) Initialize(ctx context.Context) (*mcp.InitializeResult, error) {
	// comment cleaned
	m.c.OnNotification(func(notification mcp.JSONRPCNotification) {
		fmt.Printf("鏀跺埌閫氱煡: %s\n", notification.Method)
	})

	// comment cleaned
	fmt.Println("姝ｅ湪鍒濆鍖栧鎴风...")
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "MCP-Go Weather Client",
		Version: "1.0.0",
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	serverInfo, err := m.c.Initialize(ctx, initRequest)
	if err != nil {
		return nil, fmt.Errorf("鍒濆鍖栧け璐? %w", err)
	}

	// comment cleaned
		serverInfo.ServerInfo.Name,
		serverInfo.ServerInfo.Version)

	return serverInfo, nil
}

// comment cleaned
	fmt.Println("姝ｅ湪鎵ц鍋ュ悍妫€鏌?..")
	if err := m.c.Ping(ctx); err != nil {
		return fmt.Errorf("鍋ュ悍妫€鏌ュけ璐? %w", err)
	}
	fmt.Println("鏈嶅姟鍣ㄦ甯歌繍琛屽苟鍝嶅簲")
	return nil
}

// comment cleaned
func (m *MCPClient) CallTool(ctx context.Context, toolName string, args map[string]any) (*mcp.CallToolResult, error) {
	callToolRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	}

	result, err := m.c.CallTool(ctx, callToolRequest)
	if err != nil {
		return nil, fmt.Errorf("璋冪敤宸ュ叿澶辫触: %w", err)
	}

	return result, nil
}

// comment cleaned
func (m *MCPClient) CallWeatherTool(ctx context.Context, city string) (*mcp.CallToolResult, error) {
	fmt.Printf("姝ｅ湪鏌ヨ鍩庡競 %s 鐨勫ぉ姘?..\n", city)

	callToolRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "get_weather",
			Arguments: map[string]any{
				"city": city,
			},
		},
	}

	result, err := m.c.CallTool(ctx, callToolRequest)
	if err != nil {
		return nil, fmt.Errorf("璋冪敤宸ュ叿澶辫触: %w", err)
	}

	return result, nil
}

// comment cleaned
func (m *MCPClient) GetToolResultText(result *mcp.CallToolResult) string {
	var text string
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			text += textContent.Text + "\n"
		}
	}
	return text
}

func (m *MCPClient) Close() {
	if m.c != nil {
		m.c.Close()
	}
}
