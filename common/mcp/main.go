package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	mcpclient "github.com/nanjiek/GopherMind/common/mcp/client"
	mcpserver "github.com/nanjiek/GopherMind/common/mcp/server"
)

func main() {
	mode := flag.String("mode", "", "运行模式: server 或 client")
	httpAddr := flag.String("http-addr", ":8081", "HTTP 服务地址")
	city := flag.String("city", "", "要查询天气的城市")
	flag.Parse()

	if *mode == "" {
		fmt.Println("Error: 必须指定 -mode (server 或 client)")
		flag.Usage()
		os.Exit(1)
	}

	if *mode == "server" {
		fmt.Println("启动 MCP 服务...")
		if err := mcpserver.StartServer(*httpAddr); err != nil {
			log.Fatalf("服务启动失败: %v", err)
		}
		return
	}

	if *mode == "client" {
		if *city == "" {
			fmt.Println("Error: client 模式必须指定 -city")
			flag.Usage()
			os.Exit(1)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		httpURL := "http://localhost:8081/mcp"
		mcpClient, err := mcpclient.NewMCPClient(httpURL)
		if err != nil {
			log.Fatalf("创建客户端失败: %v", err)
		}
		defer mcpClient.Close()

		if _, err := mcpClient.Initialize(ctx); err != nil {
			log.Fatalf("初始化失败: %v", err)
		}

		if err := mcpClient.Ping(ctx); err != nil {
			log.Fatalf("健康检查失败: %v", err)
		}

		result, err := mcpClient.CallWeatherTool(ctx, *city)
		if err != nil {
			log.Fatalf("调用工具失败: %v", err)
		}

		fmt.Println("\n天气查询结果:")
		fmt.Println(mcpClient.GetToolResultText(result))
		return
	}

	fmt.Println("Error: -mode 仅支持 server 或 client")
	os.Exit(1)
}
