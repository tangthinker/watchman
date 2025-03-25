package backup

import "time"

// BackupTask represents a backup task
type BackupTask struct {
	Name       string    `json:"name"`
	SourcePath string    `json:"source_path"`
	TargetPath string    `json:"target_path"`
	Schedule   string    `json:"schedule"`
	Status     string    `json:"status"`
	Progress   float64   `json:"progress"`
	LastBackup time.Time `json:"last_backup"`
	Error      string    `json:"error,omitempty"`
}
