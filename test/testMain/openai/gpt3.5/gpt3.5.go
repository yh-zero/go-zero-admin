package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const openaiAPIURL = ""
const apiKey = "sk-"

func main() {
	message := "Tell me a joke."

	// 发送聊天请求到 OpenAI 的API
	response, err := sendGPTRequest(apiKey, message)
	if err != nil {
		fmt.Println("Error sending request to GPT-3.5 API:", err)
		return
	}

	// 解析并打印生成的文本
	outputText, err := parseGPTResponse(response)
	if err != nil {
		fmt.Println("Error parsing GPT-3.5 response:", err)
		return
	}

	fmt.Println("Generated Text:", outputText)
}

func sendGPTRequest(apiKey string, inputText string) ([]byte, error) {
	client := &http.Client{
		Timeout: time.Second * 10, // 设置为 10 秒的超时
	}

	requestData := map[string]interface{}{
		"prompt":     inputText,
		"max_tokens": 100,
	}

	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", openaiAPIURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+apiKey)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	return ioutil.ReadAll(response.Body)
}

func parseGPTResponse(responseData []byte) (string, error) {
	var jsonResponse map[string]interface{}

	err := json.Unmarshal(responseData, &jsonResponse)
	if err != nil {
		return "", err
	}

	if output, ok := jsonResponse["choices"].([]interface{}); ok && len(output) > 0 {
		if text, ok := output[0].(map[string]interface{})["text"].(string); ok {
			return text, nil
		}
	}

	return "", fmt.Errorf("Unable to parse GPT-3.5 response")
}
