package main

import (
	"fmt"
	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	fmt.Println("1111111111")
	a, _ := gormadapter.NewAdapter("mysql", "root:123456@tcp(127.0.0.1:3306)/casbinRBAC", true) // Your driver and data source.

	e, err := casbin.NewEnforcer("test/casbin_RBAC-with-domainste_nants/model.conf", a)
	if err != nil {
		fmt.Println("NewEnforcer err:", err)
		panic(err)
	}

	//added, err := e.AddPolicy("yanghao", "nanli", "data2", "read") // 增加
	//fmt.Println("added:", added, "  err:", err)

	//removed, _ := e.RemovePolicy("zhangsan", "data2", "read") // 删除
	//updated, err := e.UpdatePolicy([]string{"zhangsan", "data3", "read"}, []string{"zhangsan", "data3", "write"}) // 修改
	//fmt.Println("removed:", removed)
	//fmt.Println("updated:", updated)

	//added, err = e.AddGroupingPolicy("yanghao", "yh", "nanli") //  向当前策略添加角色继承规则。
	//fmt.Println("added:", added, "  err:", err)

	//filteredPolicy := e.GetFilteredPolicy(0, "zhangsan")
	//filteredPolicy1 := e.GetFilteredPolicy(1, "data3")
	//fmt.Println("filteredPolicy:", filteredPolicy)
	//fmt.Println("filteredPolicy1:", filteredPolicy1)

	//ok, err := e.Enforce("yh", "nanli", "data2", "read") // 决定“主体”是否能够用“动作”操作访问“对象”
	ok, err := e.Enforce("yanghao", "nanli", "data2", "read") // 决定“主体”是否能够用“动作”操作访问“对象”

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
