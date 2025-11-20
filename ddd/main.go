package main

import (
	"ddd-example/cmd"
	"flag"
	"fmt"
)

func main() {
	// 解析命令行参数
	var port string
	flag.StringVar(&port, "port", "8080", "Server port")
	flag.Parse()
	
	// 创建并运行应用
	app := cmd.NewApp()
	
	fmt.Println("🚀 Starting DDD Example Application...")
	fmt.Println("📖 This example demonstrates Domain-Driven Design patterns in Go")
	fmt.Println("🔧 Features: Entities, Value Objects, Domain Services, Application Services, Repositories")
	fmt.Println()
	
	app.Run(port)
}