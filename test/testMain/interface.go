package main

import (
	"fmt"
	"reflect"
)

func main() {
	var data interface{}

	data = map[string]interface{}{
		"res": "111",
	}

	if containsResultKey(data) {
		fmt.Println("包含了")
	} else {
		fmt.Println("不包含")

	}
}

// containsResultKey 检查interface]类型的数据是否包含key“result"
func containsResultKey(data interface{}) bool {
	val := reflect.ValueOf(data)

	if val.Kind() == reflect.Map {
		for _, key := range val.MapKeys() {
			if key.String() == "res" {
				return true
			}
		}
	}
	return false
}
