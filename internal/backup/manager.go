package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/tangthinker/watchman/internal/config"
)

// Manager 管理所有备份任务
type Manager struct {
	config     *config.Config
	cron       *cron.Cron
	tasks      map[string]*config.BackupTask
	configFile string
	mu         sync.RWMutex
}

// NewManager 创建一个新的备份管理器
func NewManager(configFile string) (*Manager, error) {
	m := &Manager{
		config:     &config.Config{},
		cron:       cron.New(cron.WithSeconds()),
		tasks:      make(map[string]*config.BackupTask),
		configFile: configFile,
	}

	// 加载配置
	if err := m.loadConfig(); err != nil {
		return nil, fmt.Errorf("failed to load config: %v", err)
	}

	// 启动所有任务
	m.startAllTasks()

	return m, nil
}

// loadConfig 从文件加载配置
func (m *Manager) loadConfig() error {
	data, err := os.ReadFile(m.configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return m.saveConfig()
		}
		return err
	}

	if err := json.Unmarshal(data, m.config); err != nil {
		return err
	}

	// 初始化任务映射
	for i := range m.config.Tasks {
		m.tasks[m.config.Tasks[i].ID] = &m.config.Tasks[i]
	}

	return nil
}

// saveConfig 保存配置到文件
func (m *Manager) saveConfig() error {
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}

	// 确保配置目录存在
	if err := os.MkdirAll(filepath.Dir(m.configFile), 0755); err != nil {
		return err
	}

	return os.WriteFile(m.configFile, data, 0644)
}

// startAllTasks 启动所有任务
func (m *Manager) startAllTasks() {
	for _, task := range m.tasks {
		if task.Status == "running" {
			m.startTask(task)
		}
	}
	m.cron.Start()
}

// startTask 启动单个任务
func (m *Manager) startTask(task *config.BackupTask) {
	_, err := m.cron.AddFunc(task.Schedule, func() {
		if err := m.runBackup(task); err != nil {
			task.Status = "error"
			task.Error = err.Error()
			m.saveConfig()
		}
	})

	if err != nil {
		task.Status = "error"
		task.Error = fmt.Sprintf("Failed to schedule task: %v", err)
		m.saveConfig()
	}
}

// runBackup 执行备份
func (m *Manager) runBackup(task *config.BackupTask) error {
	task.Status = "running"
	task.Progress = 0
	m.saveConfig()

	// 创建进度通道
	progressChan := make(chan float64)

	// 在单独的 goroutine 中执行同步
	errChan := make(chan error, 1)
	go func() {
		errChan <- Sync(task.SourcePath, task.TargetPath, progressChan)
	}()

	// 更新进度
	for {
		select {
		case progress := <-progressChan:
			task.Progress = progress
			m.saveConfig()
		case err := <-errChan:
			close(progressChan)
			if err != nil {
				task.Status = "error"
				task.Error = err.Error()
				task.Progress = 0
			} else {
				task.Status = "running"
				task.Progress = 100
				task.LastBackup = time.Now()
				task.Error = ""
			}
			return m.saveConfig()
		}
	}
}

// AddTask 添加新的备份任务
func (m *Manager) AddTask(task config.BackupTask) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tasks[task.ID]; exists {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}

	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	task.Status = "running"

	m.config.Tasks = append(m.config.Tasks, task)
	m.tasks[task.ID] = &task

	if err := m.saveConfig(); err != nil {
		return err
	}

	m.startTask(&task)
	return nil
}

// ListTasks 列出所有备份任务
func (m *Manager) ListTasks() []config.BackupTask {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := make([]config.BackupTask, 0, len(m.tasks))
	for _, task := range m.tasks {
		tasks = append(tasks, *task)
	}
	return tasks
}

// StopTask 停止指定的备份任务
func (m *Manager) StopTask(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task with ID %s not found", taskID)
	}

	task.Status = "stopped"
	task.UpdatedAt = time.Now()
	return m.saveConfig()
}

// DeleteTask 删除指定的备份任务
func (m *Manager) DeleteTask(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tasks[taskID]; !exists {
		return fmt.Errorf("task with ID %s not found", taskID)
	}

	// 从任务列表中删除
	for i, task := range m.config.Tasks {
		if task.ID == taskID {
			m.config.Tasks = append(m.config.Tasks[:i], m.config.Tasks[i+1:]...)
			break
		}
	}

	delete(m.tasks, taskID)
	return m.saveConfig()
}
