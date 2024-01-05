package main

import (
	"fmt"
	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	a, _ := gormadapter.NewAdapter("mysql", "root:123456@tcp(127.0.0.1:3306)/casbin", true) // Your driver and data source.
	e, _ := casbin.NewEnforcer("./model.conf", a)

	e.AddFunction("my_func", KeyMatchFunc)

	//added, err := e.AddPolicy("zhangsan", "data2", "read")    // 增加
	//removed, _ := e.RemovePolicy("zhangsan", "data2", "read") // 删除
	//updated, err := e.UpdatePolicy([]string{"zhangsan", "data3", "read"}, []string{"zhangsan", "data3", "write"}) // 修改
	//fmt.Println("added:", added, "  err:", err)
	//fmt.Println("removed:", removed)
	//fmt.Println("updated:", updated)

	//added, err := e.AddGroupingPolicy("group1", "data2_admin") //  向当前策略添加角色继承规则。
	//fmt.Println("added:", added, "  err:", err)

	//filteredPolicy := e.GetFilteredPolicy(0, "zhangsan")
	//filteredPolicy1 := e.GetFilteredPolicy(1, "data3")
	//fmt.Println("filteredPolicy:", filteredPolicy)
	//fmt.Println("filteredPolicy1:", filteredPolicy1)

	ok, err := e.Enforce("data2_admin", "data3", "read") // 决定“主体”是否能够用“动作”操作访问“对象”

	if err != nil {
		// 处理err
		fmt.Printf(" 错误 err:%v", err)
	}
	if ok == true {
		// 允许alice读取data1
		fmt.Println("通过")
	} else {
		// 拒绝请求，抛出异常
		fmt.Println("不通过")
	}
}

func KeyMatch(key1 string, key2 string) bool {
	//fmt.Println("key1 :", key1)
	//fmt.Println("key2 :", key2)
	//return key1 != key2
	return key1 == key2

	//return key1 != "data1" && key1 == key2
}

func KeyMatchFunc(args ...interface{}) (interface{}, error) {
	name1 := args[0].(string)
	name2 := args[1].(string)

	return (bool)(KeyMatch(name1, name2)), nil
}
