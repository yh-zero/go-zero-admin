package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

const apiKey = ""
const apiUrl = ""

type GPTRequest struct {
	Prompt    string `json:"prompt"`
	MaxTokens int    `json:"max_tokens"`
}

type GPTResponse struct {
	Choices []struct {
		Text string `json:"text"`
	} `json:"choices"`
}

func main() {
	// 构造 GPT API 请求数据
	rawPrompt := "Translate the following English text to French: 'Hello, how are you?'"
	cleanedPrompt := cleanJSONString(rawPrompt)

	requestData := GPTRequest{
		Prompt:    cleanedPrompt,
		MaxTokens: 150,
	}

	// 转换请求数据为 JSON 格式
	requestBody, err := json.Marshal(requestData)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}

	// 发送 HTTP POST 请求到 OpenAI API
	client := &http.Client{}
	req, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Println("Error creating HTTP request:", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	response, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending HTTP request:", err)
		return
	}
	defer response.Body.Close()

	// 读取响应体
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	// 解析 JSON 响应
	var gptResponse GPTResponse
	err = json.Unmarshal(responseBody, &gptResponse)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return
	}

	// 输出 GPT 生成的文本
	if len(gptResponse.Choices) > 0 {
		generatedText := gptResponse.Choices[0].Text
		fmt.Println("Generated Text:", generatedText)
	} else {
		fmt.Println("No generated text found.")
	}
}

// 清理字符串中可能导致 JSON 错误的字符
func cleanJSONString(s string) string {
	// 在这里，你可以添加代码以清理字符串，确保它不包含 JSON 不支持的字符
	// 这个示例中，简单地移除字符串中的换行符
	return strings.ReplaceAll(s, "\n", " ")
}
