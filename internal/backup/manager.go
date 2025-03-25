package backup

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Manager manages backup tasks
type Manager struct {
	configFile string
	tasks      map[string]*BackupTask
	timers     map[string]*time.Timer
	mu         sync.RWMutex
}

// NewManager creates a new backup manager
func NewManager(configFile string) (*Manager, error) {
	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configFile)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %v", err)
	}

	manager := &Manager{
		configFile: configFile,
		tasks:      make(map[string]*BackupTask),
		timers:     make(map[string]*time.Timer),
	}

	// Load existing tasks
	if err := manager.loadTasks(); err != nil {
		log.Printf("Warning: failed to load tasks: %v", err)
	}

	return manager, nil
}

// AddTask adds a new backup task
func (m *Manager) AddTask(task BackupTask) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Printf("Adding task to manager: %+v", task)

	// 重新加载任务列表，确保数据是最新的
	if err := m.loadTasks(); err != nil {
		log.Printf("Warning: failed to reload tasks: %v", err)
	}

	// Check if task already exists
	if _, exists := m.tasks[task.Name]; exists {
		return fmt.Errorf("task %s already exists", task.Name)
	}

	// Initialize task status
	task.Status = "Ready"
	task.Progress = 100 // 初始状态为 Ready 时，进度应该是 100%
	task.LastBackup = time.Time{}

	// Store task
	m.tasks[task.Name] = &task

	log.Printf("Starting backup timer for task: %s", task.Name)
	// Start backup timer
	if err := m.startBackupTimer(task.Name); err != nil {
		delete(m.tasks, task.Name)
		return fmt.Errorf("failed to start backup timer: %v", err)
	}

	log.Printf("Saving tasks to file")
	// Save tasks to file
	if err := m.saveTasks(); err != nil {
		delete(m.tasks, task.Name)
		m.stopBackupTimer(task.Name)
		return fmt.Errorf("failed to save tasks: %v", err)
	}

	log.Printf("Task added successfully: %s", task.Name)
	return nil
}

// ListTasks returns all backup tasks
func (m *Manager) ListTasks() []BackupTask {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// // 重新加载任务列表，确保数据是最新的
	// if err := m.loadTasks(); err != nil {
	// 	log.Printf("Warning: failed to reload tasks: %v", err)
	// }

	// log.Printf("Listing %d tasks", len(m.tasks))
	tasks := make([]BackupTask, 0, len(m.tasks))
	for _, task := range m.tasks {
		tasks = append(tasks, *task)
	}
	return tasks
}

// DeleteTask deletes a backup task
func (m *Manager) DeleteTask(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if task exists
	if _, exists := m.tasks[name]; !exists {
		return fmt.Errorf("task %s does not exist", name)
	}

	// Stop backup timer
	m.stopBackupTimer(name)

	// Delete task
	delete(m.tasks, name)

	// Save tasks to file
	if err := m.saveTasks(); err != nil {
		return fmt.Errorf("failed to save tasks: %v", err)
	}

	return nil
}

// StopTask stops a backup task
func (m *Manager) StopTask(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if task exists
	task, exists := m.tasks[name]
	if !exists {
		return fmt.Errorf("task %s does not exist", name)
	}

	// Stop backup timer
	m.stopBackupTimer(name)

	// Update task status
	task.Status = "Stopped"
	task.Progress = 0 // 停止时设置为 0

	// Save tasks to file
	if err := m.saveTasks(); err != nil {
		return fmt.Errorf("failed to save tasks: %v", err)
	}

	return nil
}

// Shutdown stops all backup timers
func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name := range m.timers {
		m.stopBackupTimer(name)
	}
}

// loadTasks loads tasks from the config file
func (m *Manager) loadTasks() error {
	// 添加日志
	log.Printf("Loading tasks from file: %s", m.configFile)

	data, err := os.ReadFile(m.configFile)
	if os.IsNotExist(err) {
		log.Printf("Config file does not exist, starting with empty task list")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	var tasks []BackupTask
	if err := json.Unmarshal(data, &tasks); err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	// 清空现有任务
	m.tasks = make(map[string]*BackupTask)

	// 添加日志
	log.Printf("Found %d tasks in config file", len(tasks))

	for _, task := range tasks {
		taskCopy := task
		m.tasks[task.Name] = &taskCopy
		if task.Status != "Stopped" {
			if err := m.startBackupTimer(task.Name); err != nil {
				log.Printf("Warning: failed to start timer for task %s: %v", task.Name, err)
			}
		}
	}

	return nil
}

// saveTasks saves tasks to the config file
func (m *Manager) saveTasks() error {
	tasks := make([]BackupTask, 0, len(m.tasks))
	for _, task := range m.tasks {
		tasks = append(tasks, *task)
	}

	log.Printf("Saving %d tasks to file: %s", len(tasks), m.configFile)
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %v", err)
	}

	// 确保配置目录存在
	configDir := filepath.Dir(m.configFile)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	// 添加文件权限检查
	if err := os.WriteFile(m.configFile, data, 0644); err != nil {
		log.Printf("Failed to write config file: %v", err)
		// 尝试检查文件权限
		if info, statErr := os.Stat(configDir); statErr == nil {
			log.Printf("Config directory permissions: %v", info.Mode())
		}
		return fmt.Errorf("failed to write config file: %v", err)
	}

	log.Printf("Successfully saved tasks to file")
	return nil
}

// startBackupTimer starts a timer for periodic backup
func (m *Manager) startBackupTimer(name string) error {
	task := m.tasks[name]
	interval, err := time.ParseDuration(task.Schedule + "m")
	if err != nil {
		return fmt.Errorf("invalid schedule: %v", err)
	}

	// 打印定时器启动日志
	log.Printf("[Task: %s] Starting backup timer with interval: %s",
		task.Name, interval.String())

	timer := time.NewTimer(interval)
	m.timers[name] = timer

	// 立即执行一次备份
	log.Printf("[Task: %s] Performing initial backup", task.Name)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[Task: %s] Backup failed: %v", task.Name, r)
			}
		}()
		if err := m.performBackup(name); err != nil {
			log.Printf("[Task: %s] Backup failed: %v", task.Name, err)
		}
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[Task: %s] Backup failed: %v", task.Name, r)
			}
		}()
		for {
			<-timer.C
			// 打印定时器触发日志
			log.Printf("[Task: %s] Timer triggered, starting backup", task.Name)

			if err := m.performBackup(name); err != nil {
				log.Printf("[Task: %s] Backup failed: %v", task.Name, err)
			}
			timer.Reset(interval)
			// 打印下次备份时间
			log.Printf("[Task: %s] Next backup scheduled at: %s",
				task.Name, time.Now().Add(interval).Format("2006-01-02 15:04:05"))
		}
	}()

	return nil
}

// stopBackupTimer stops a backup timer
func (m *Manager) stopBackupTimer(name string) {
	if timer, exists := m.timers[name]; exists {
		timer.Stop()
		delete(m.timers, name)
		// 打印停止日志
		log.Printf("[Task: %s] Backup timer stopped", name)
	}
}

// performBackup performs the actual backup operation
func (m *Manager) performBackup(name string) error {
	m.mu.Lock()
	task := m.tasks[name]
	if task == nil {
		m.mu.Unlock()
		return fmt.Errorf("task %s does not exist", name)
	}

	log.Printf("[Task: %s] Starting backup from %s to %s",
		task.Name, task.SourcePath, task.TargetPath)

	task.Status = "Running"
	task.Progress = 0 // 开始备份时设置为 0
	task.Error = ""
	m.mu.Unlock()

	// TODO: Implement actual backup logic here
	// For now, just simulate a backup operation
	// for i := 0; i <= 100; i += 10 {
	// 	time.Sleep(100 * time.Millisecond)
	// 	m.mu.Lock()
	// 	task.Progress = float64(i)
	// 	log.Printf("[Task: %s] Progress: %.1f%%", task.Name, task.Progress)
	// 	m.mu.Unlock()
	// }

	progressChan := make(chan float64)
	errChan := make(chan error)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[Task: %s] Backup failed: %v", task.Name, r)
			}
		}()
		errChan <- Sync(task.SourcePath, task.TargetPath, progressChan)
		close(progressChan)
		close(errChan)
	}()

outer:
	for {
		select {
		case err := <-errChan:
			if err != nil {
				log.Printf("[Task: %s] Backup failed: %v", task.Name, err)
			}
			break outer
		case progress := <-progressChan:
			log.Printf("[Task: %s] Progress: %.1f%%", task.Name, progress)
			m.mu.Lock()
			task.Progress = progress
			m.mu.Unlock()
		}
	}

	m.mu.Lock()
	task.Status = "Ready"
	task.Progress = 100 // 完成备份时设置为 100
	task.LastBackup = time.Now()
	log.Printf("[Task: %s] Backup completed successfully at %s",
		task.Name, task.LastBackup.Format("2006-01-02 15:04:05"))
	m.mu.Unlock()

	return nil
}
