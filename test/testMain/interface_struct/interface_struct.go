package main

import (
	"fmt"
)

type IsResult struct {
	Result string `json:"result"`
}

type RandomImageResponse struct {
	Result string `json:"result"`
}

func Success(data interface{}) {
	fmt.Printf("data: %+v\n", data) // 使用 %+v 输出结构体的完整 key-value 对

	if d, ok := data.(*IsResult); ok {
		fmt.Printf("Result: %+v\n", d) // 使用 %+v 输出结构体的完整 key-value 对
	} else {
		fmt.Println("类型断言失败，期望 *IsResult 类型")
	}

}

func main() {
	// 传递指针类型的数据
	var data interface{} = &RandomImageResponse{
		Result: "111",
	}

	Success(data)
}
