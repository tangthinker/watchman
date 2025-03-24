package config

import (
	"time"
)

// BackupTask 表示一个备份任务
type BackupTask struct {
	ID         string    `json:"id"`          // 任务ID
	Name       string    `json:"name"`        // 任务名称
	SourcePath string    `json:"source_path"` // 源目录路径
	TargetPath string    `json:"target_path"` // 目标目录路径
	Schedule   string    `json:"schedule"`    // cron表达式
	LastBackup time.Time `json:"last_backup"` // 上次备份时间
	Status     string    `json:"status"`      // 任务状态：running, stopped, error
	Progress   float64   `json:"progress"`    // 当前进度（0-100）
	Error      string    `json:"error"`       // 错误信息
	CreatedAt  time.Time `json:"created_at"`  // 创建时间
	UpdatedAt  time.Time `json:"updated_at"`  // 更新时间
}

// Config 存储所有备份任务的配置
type Config struct {
	Tasks []BackupTask `json:"tasks"`
}
