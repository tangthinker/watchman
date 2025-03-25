package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/tangthinker/watchman/internal/backup"
	"github.com/tangthinker/watchman/internal/client"
	"github.com/tangthinker/watchman/internal/daemon"
)

var (
	configFile = flag.String("config", filepath.Join(os.Getenv("HOME"), ".watchman", "config.json"), "配置文件路径")
	interval   = flag.Int("n", 0, "备份间隔（分钟）")
)

// 检查是否已有守护进程在运行
func checkRunningDaemon() bool {
	output, err := os.ReadFile("/tmp/watchman.pid")
	if err != nil {
		return false
	}

	pid := strings.TrimSpace(string(output))
	if pid == "" {
		return false
	}

	// 尝试向进程发送信号0来检查进程是否存在
	pidNum := 0
	fmt.Sscanf(pid, "%d", &pidNum)
	if pidNum <= 0 {
		os.Remove("/tmp/watchman.pid")
		return false
	}

	process, err := os.FindProcess(pidNum)
	if err != nil {
		os.Remove("/tmp/watchman.pid")
		return false
	}

	// 在Unix系统中，发送信号0用于检查进程是否存在
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		os.Remove("/tmp/watchman.pid")
		return false
	}

	return true
}

// 创建进程锁
func createPIDFile() error {
	pid := fmt.Sprintf("%d", os.Getpid())
	return os.WriteFile("/tmp/watchman.pid", []byte(pid), 0644)
}

// 清理进程锁
func cleanupPIDFile() {
	os.Remove("/tmp/watchman.pid")
}

func main() {
	// 解析命令行参数
	flag.Parse()

	// 如果有命令行参数，作为客户端运行
	if len(flag.Args()) > 0 {
		handleClientCommand()
		return
	}

	// 否则作为守护进程运行
	runAsDaemon()
}

func handleClientCommand() {
	// 创建客户端连接
	c, err := client.NewClient()
	if err != nil {
		log.Fatalf("Failed to connect to daemon: %v", err)
	}
	defer c.Close()

	// 添加调试日志
	log.Printf("Connected to daemon, sending command: %s", flag.Arg(0))

	// 处理命令
	switch flag.Arg(0) {
	case "add":
		if len(flag.Args()) != 4 {
			fmt.Println("Usage: watchman -n <minutes> add <name> <source_path> <target_path>")
			fmt.Println("Note: The -n flag must come before the 'add' command")
			os.Exit(1)
		}

		if *interval <= 0 {
			fmt.Println("Error: interval (-n) must be greater than 0")
			os.Exit(1)
		}

		log.Printf("Adding task: name=%s, source=%s, target=%s, interval=%d",
			flag.Arg(1), flag.Arg(2), flag.Arg(3), *interval)

		err = c.AddTask(
			flag.Arg(1),                  // name
			flag.Arg(2),                  // source_path
			flag.Arg(3),                  // target_path
			fmt.Sprintf("%d", *interval), // schedule
		)
		if err != nil {
			log.Fatalf("Failed to add task: %v", err)
		}
		log.Printf("Task added successfully")

	case "list":
		tasks, err := c.ListTasks()
		if err == nil {
			printTasks(tasks)
			return
		}

	case "delete":
		if len(flag.Args()) != 2 {
			fmt.Println("Usage: watchman delete <task_name>")
			os.Exit(1)
		}
		err = c.DeleteTask(flag.Arg(1))

	case "stop":
		if len(flag.Args()) != 2 {
			fmt.Println("Usage: watchman stop <task_name>")
			os.Exit(1)
		}
		err = c.StopTask(flag.Arg(1))

	default:
		fmt.Println("Available commands:")
		fmt.Println("  watchman -n <minutes> add <name> <source_path> <target_path> - Add a new backup task")
		fmt.Println("  watchman list - List all backup tasks")
		fmt.Println("  watchman stop <task_name> - Stop a backup task")
		fmt.Println("  watchman delete <task_name> - Delete a backup task")
		fmt.Println("\nNote: When using flags (like -n), they must come before the command")
		os.Exit(1)
	}

	if err != nil {
		log.Fatalf("Command failed: %v", err)
	}
}

func printTasks(tasks interface{}) {
	// 首先尝试将 interface{} 转换为 []interface{}
	taskList, ok := tasks.([]interface{})
	if !ok {
		log.Printf("Failed to convert tasks: %T", tasks)
		fmt.Println("No backup tasks found")
		return
	}

	if len(taskList) == 0 {
		fmt.Println("No backup tasks found")
		return
	}

	// 定义表格格式
	format := "%-20s\t%-30s\t%-30s\t%-10s\t%-10s\t%-10s\t%-25s\n"

	// 打印表头
	fmt.Printf(format, "NAME", "SOURCE", "TARGET", "INTERVAL", "STATUS", "PROGRESS", "LAST BACKUP")

	// 打印任务信息
	for _, t := range taskList {
		// 将每个任务转换为 map[string]interface{}
		task, ok := t.(map[string]interface{})
		if !ok {
			log.Printf("Failed to convert task to map: %T", t)
			continue
		}

		// 安全地获取字段值
		name := getStringValue(task, "name")
		sourcePath := getStringValue(task, "source_path")
		targetPath := getStringValue(task, "target_path")
		schedule := getStringValue(task, "schedule")
		status := getStringValue(task, "status")
		progress := getFloatValue(task, "progress")
		lastBackup := getStringValue(task, "last_backup")
		if lastBackup == "" {
			lastBackup = "-"
		}

		// 如果路径太长，截断并添加...
		if len(sourcePath) > 27 {
			sourcePath = sourcePath[:24] + "..."
		}
		if len(targetPath) > 27 {
			targetPath = targetPath[:24] + "..."
		}

		fmt.Printf(format,
			name,
			sourcePath,
			targetPath,
			schedule+"m",
			status,
			fmt.Sprintf("%.1f%%", progress),
			lastBackup,
		)

		// 如果有错误，在下一行显示
		if errStr := getStringValue(task, "error"); errStr != "" {
			fmt.Printf("  Error: %s\n", errStr)
		}
	}
}

// 辅助函数：安全地获取字符串值
func getStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// 辅助函数：安全地获取浮点数值
func getFloatValue(m map[string]interface{}, key string) float64 {
	switch v := m[key].(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

func runAsDaemon() {
	// 检查是否已有守护进程在运行
	if checkRunningDaemon() {
		log.Fatal("Watchman daemon is already running")
	}

	// 创建进程锁
	if err := createPIDFile(); err != nil {
		log.Fatalf("Failed to create PID file: %v", err)
	}
	defer cleanupPIDFile()

	// 创建备份管理器
	manager, err := backup.NewManager(*configFile)
	if err != nil {
		log.Fatalf("Failed to create backup manager: %v", err)
	}

	// 创建并启动 socket 服务器
	server, err := daemon.NewServer(manager)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	defer server.Close()

	// 处理信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动服务器
	go func() {
		if err := server.Start(); err != nil {
			log.Printf("Server error: %v", err)
			sigChan <- syscall.SIGTERM
		}
	}()

	log.Println("Watchman daemon started")

	// 等待信号
	<-sigChan

	// 关闭所有定时器
	manager.Shutdown()
	log.Println("Shutting down Watchman...")
}
