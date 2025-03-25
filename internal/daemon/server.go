package daemon

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/tangthinker/watchman/internal/backup"
	"github.com/tangthinker/watchman/internal/ipc"
)

type Server struct {
	listener net.Listener
	manager  *backup.Manager
}

// NewServer creates a new Unix domain socket server
func NewServer(manager *backup.Manager) (*Server, error) {
	// Remove existing socket file if it exists
	if err := os.RemoveAll(ipc.SockAddr); err != nil {
		return nil, fmt.Errorf("failed to remove existing socket: %v", err)
	}

	// Create Unix domain socket
	listener, err := net.Listen("unix", ipc.SockAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create socket: %v", err)
	}

	// Set socket file permissions
	if err := os.Chmod(ipc.SockAddr, 0666); err != nil {
		listener.Close()
		return nil, fmt.Errorf("failed to set socket permissions: %v", err)
	}

	return &Server{
		listener: listener,
		manager:  manager,
	}, nil
}

// Start starts the server and handles incoming connections
func (s *Server) Start() error {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return fmt.Errorf("failed to accept connection: %v", err)
		}
		go s.handleConnection(conn)
	}
}

// Close closes the server
func (s *Server) Close() error {
	if err := s.listener.Close(); err != nil {
		return fmt.Errorf("failed to close listener: %v", err)
	}
	return os.RemoveAll(ipc.SockAddr)
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Read command
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		log.Printf("Failed to read from connection: %v", err)
		return
	}

	// Parse command
	cmd, err := ipc.UnmarshalCommand(buf[:n])
	if err != nil {
		sendError(conn, fmt.Errorf("invalid command: %v", err))
		return
	}

	// Handle command
	var resp *ipc.Response
	switch cmd.Type {
	case ipc.CmdAdd:
		resp = s.handleAdd(cmd.Payload)
	case ipc.CmdList:
		resp = s.handleList()
	case ipc.CmdDelete:
		resp = s.handleDelete(cmd.Payload)
	case ipc.CmdStop:
		resp = s.handleStop(cmd.Payload)
	default:
		resp = ipc.NewResponse(false, nil, fmt.Errorf("unknown command type: %s", cmd.Type))
	}

	// Send response
	if data, err := resp.Marshal(); err != nil {
		log.Printf("Failed to marshal response: %v", err)
	} else if _, err := conn.Write(data); err != nil {
		log.Printf("Failed to send response: %v", err)
	}
}

func (s *Server) handleAdd(payload map[string]any) *ipc.Response {
	name, _ := payload["name"].(string)
	sourcePath, _ := payload["source_path"].(string)
	targetPath, _ := payload["target_path"].(string)
	schedule, _ := payload["schedule"].(string)

	log.Printf("Received add task request: name=%s, source=%s, target=%s, schedule=%s",
		name, sourcePath, targetPath, schedule)

	if name == "" || sourcePath == "" || targetPath == "" || schedule == "" {
		return ipc.NewResponse(false, nil, fmt.Errorf("missing required fields"))
	}

	task := backup.BackupTask{
		Name:       name,
		SourcePath: sourcePath,
		TargetPath: targetPath,
		Schedule:   schedule,
	}

	err := s.manager.AddTask(task)
	if err != nil {
		log.Printf("Failed to add task: %v", err)
		return ipc.NewResponse(false, nil, err)
	}

	log.Printf("Task added successfully")
	return ipc.NewResponse(true, nil, nil)
}

func (s *Server) handleList() *ipc.Response {
	tasks := s.manager.ListTasks()

	// 将任务转换为map以便JSON序列化
	taskMaps := make([]map[string]interface{}, len(tasks))
	for i, task := range tasks {
		taskMaps[i] = map[string]interface{}{
			"name":        task.Name,
			"source_path": task.SourcePath,
			"target_path": task.TargetPath,
			"schedule":    task.Schedule,
			"status":      task.Status,
			"progress":    task.Progress,
			"last_backup": task.LastBackup.Format("2006-01-02 15:04:05"),
			"error":       task.Error,
		}
	}

	return ipc.NewResponse(true, taskMaps, nil)
}

func (s *Server) handleDelete(payload map[string]any) *ipc.Response {
	name, _ := payload["name"].(string)
	if name == "" {
		return ipc.NewResponse(false, nil, fmt.Errorf("task name is required"))
	}

	err := s.manager.DeleteTask(name)
	return ipc.NewResponse(err == nil, nil, err)
}

func (s *Server) handleStop(payload map[string]any) *ipc.Response {
	name, _ := payload["name"].(string)
	if name == "" {
		return ipc.NewResponse(false, nil, fmt.Errorf("task name is required"))
	}

	err := s.manager.StopTask(name)
	return ipc.NewResponse(err == nil, nil, err)
}

func sendError(conn net.Conn, err error) {
	resp := ipc.NewResponse(false, nil, err)
	if data, err := resp.Marshal(); err == nil {
		conn.Write(data)
	}
}
