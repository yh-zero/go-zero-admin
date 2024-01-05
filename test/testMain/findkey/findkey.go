package main

import "fmt"

type m map[string]struct{}

func (m m) findKey(key string) bool {
	if _, ok := m[key]; ok {
		return true
	}
	return false
}

//func (m m) insert(key string) {}

func main() {
	var mm m = make(m, 10)
	mm["result"] = struct{}{}

	fmt.Println(isContainKey(mm, "result"))
	fmt.Println(isContainKey(mm, "xxx"))
}

func isContainKey(intf interface {
	findKey(key string) bool
}, key string) bool {
	return intf.findKey(key)
}
