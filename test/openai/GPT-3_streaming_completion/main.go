package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"

	openai "github.com/openai/openai-go/v2"
)

func main() {
	apiKey := "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"
	proxyPort := "https://gpt.jwhma.top/surirui666/v1/chat/completions"

	// 创建默认配置
	config := openai.DefaultConfig(apiKey)

	// 设置代理
	//proxyURL := fmt.Sprintf("http://localhost:%s", proxyPort)
	proxyURL := https://gpt.jwhma.top/surirui666/v1/chat/completions
	proxy, err := url.Parse(proxyURL)
	if err != nil {
		panic(err)
	}
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxy),
	}
	config.HTTPClient = &http.Client{
		Transport: transport,
	}

	// 创建 OpenAI 客户端
	client := openai.NewClientWithConfig(config)

	// 发起 ChatGPT 流式完成请求
	stream := client.StreamingCreateChat(nil)

	// 发送请求
	if err := stream.Send(&openai.ChatRequest{
		Documents:   []string{"Once upon a time"},
		MaxTokens:   50,
		Stop:        "\n",
		Temperature: 0.7,
	}); err != nil {
		fmt.Println("Error sending request:", err)
		os.Exit(1)
	}

	// 接收并打印响应
	for {
		resp, err := stream.Recv()
		if err != nil {
			fmt.Println("Error receiving response:", err)
			os.Exit(1)
		}

		fmt.Println("Received response:", resp)
	}
}
