package main

import (
	"fmt"
	"github.com/robfig/cron/v3"
)

func myScheduledTask() {
	fmt.Println("定时任务执行了！")
}

func main() {
	// 创建 cron 实例
	c := cron.New()

	// 添加定时任务，每分钟执行一次
	_, err := c.AddFunc("@every 1s", myScheduledTask)
	if err != nil {
		fmt.Println("添加定时任务失败:", err)
		return
	}

	// 启动 cron
	c.Start()

	// 等待程序运行，以便定时任务有足够时间执行
	select {}
}
