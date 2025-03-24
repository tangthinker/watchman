package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/tangthinker/watchman/internal/backup"
	"github.com/tangthinker/watchman/internal/config"
)

var (
	configFile = flag.String("config", filepath.Join(os.Getenv("HOME"), ".watchman", "config.json"), "配置文件路径")
	interval   = flag.Int("n", 0, "备份间隔（分钟），如果指定此参数，将忽略schedule参数")
)

func main() {
	// 解析命令行参数
	flag.Parse()

	// 创建备份管理器
	manager, err := backup.NewManager(*configFile)
	if err != nil {
		log.Fatalf("Failed to create backup manager: %v", err)
	}

	// 处理命令行参数
	if len(flag.Args()) > 0 {
		handleCommand(manager, flag.Args())
		return
	}

	// 作为守护进程运行
	runAsDaemon(manager)
}

func handleCommand(manager *backup.Manager, args []string) {
	switch args[0] {
	case "add":
		var task config.BackupTask
		if *interval > 0 {
			// 使用-n参数模式
			if len(args) != 4 {
				fmt.Println("Usage: watchman add -n <minutes> <name> <source_path> <target_path>")
				os.Exit(1)
			}
			task = config.BackupTask{
				ID:         uuid.New().String(),
				Name:       args[1],
				SourcePath: args[2],
				TargetPath: args[3],
				Schedule:   fmt.Sprintf("0 */%d * * * *", *interval),
			}
		} else {
			// 使用schedule参数模式
			if len(args) != 5 {
				fmt.Println("Usage: watchman add <name> <source_path> <target_path> <schedule>")
				fmt.Println("Example: watchman add mybackup /source/dir /target/dir \"0 */1 * * * *\"")
				os.Exit(1)
			}
			task = config.BackupTask{
				ID:         uuid.New().String(),
				Name:       args[1],
				SourcePath: args[2],
				TargetPath: args[3],
				Schedule:   args[4],
			}
		}

		if err := manager.AddTask(task); err != nil {
			log.Fatalf("Failed to add task: %v", err)
		}
		fmt.Printf("Task %s added successfully\n", task.Name)
		fmt.Printf("Schedule: %s\n", task.Schedule)

	case "list":
		tasks := manager.ListTasks()
		if len(tasks) == 0 {
			fmt.Println("No backup tasks found")
			return
		}
		fmt.Println("Backup Tasks:")
		for _, task := range tasks {
			fmt.Printf("\nID: %s\n", task.ID)
			fmt.Printf("Name: %s\n", task.Name)
			fmt.Printf("Source: %s\n", task.SourcePath)
			fmt.Printf("Target: %s\n", task.TargetPath)
			fmt.Printf("Schedule: %s\n", task.Schedule)
			fmt.Printf("Status: %s\n", task.Status)
			fmt.Printf("Progress: %.2f%%\n", task.Progress)
			if task.LastBackup.Unix() > 0 {
				fmt.Printf("Last Backup: %s\n", task.LastBackup.Format(time.RFC3339))
			}
			if task.Error != "" {
				fmt.Printf("Error: %s\n", task.Error)
			}
		}

	case "stop":
		if len(args) != 2 {
			fmt.Println("Usage: watchman stop <task_id>")
			os.Exit(1)
		}
		if err := manager.StopTask(args[1]); err != nil {
			log.Fatalf("Failed to stop task: %v", err)
		}
		fmt.Println("Task stopped successfully")

	case "delete":
		if len(args) != 2 {
			fmt.Println("Usage: watchman delete <task_id>")
			os.Exit(1)
		}
		if err := manager.DeleteTask(args[1]); err != nil {
			log.Fatalf("Failed to delete task: %v", err)
		}
		fmt.Println("Task deleted successfully")

	default:
		fmt.Println("Available commands:")
		fmt.Println("  add <name> <source_path> <target_path> <schedule> - Add a new backup task with cron schedule")
		fmt.Println("  add -n <minutes> <name> <source_path> <target_path> - Add a new backup task with interval in minutes")
		fmt.Println("  list - List all backup tasks")
		fmt.Println("  stop <task_id> - Stop a backup task")
		fmt.Println("  delete <task_id> - Delete a backup task")
		os.Exit(1)
	}
}

func runAsDaemon(manager *backup.Manager) {
	log.Println("Starting Watchman daemon...")

	// 处理信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待信号
	<-sigChan
	log.Println("Shutting down Watchman...")
}
