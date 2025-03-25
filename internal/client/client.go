package client

import (
	"fmt"
	"net"

	"github.com/tangthinker/watchman/internal/ipc"
)

type Client struct {
	conn net.Conn
}

// NewClient creates a new Unix domain socket client
func NewClient() (*Client, error) {
	conn, err := net.Dial("unix", ipc.SockAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %v", err)
	}

	return &Client{conn: conn}, nil
}

// Close closes the client connection
func (c *Client) Close() error {
	return c.conn.Close()
}

// SendCommand sends a command to the daemon and returns the response
func (c *Client) SendCommand(cmd *ipc.Command) (*ipc.Response, error) {
	// Marshal and send command
	data, err := cmd.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command: %v", err)
	}

	if _, err := c.conn.Write(data); err != nil {
		return nil, fmt.Errorf("failed to send command: %v", err)
	}

	// Read response
	buf := make([]byte, 4096)
	n, err := c.conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Unmarshal response
	resp, err := ipc.UnmarshalResponse(buf[:n])
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return resp, nil
}

// AddTask sends an add task command to the daemon
func (c *Client) AddTask(name, sourcePath, targetPath, schedule string) error {
	cmd := ipc.NewCommand(ipc.CmdAdd, map[string]any{
		"name":        name,
		"source_path": sourcePath,
		"target_path": targetPath,
		"schedule":    schedule,
	})

	resp, err := c.SendCommand(cmd)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf(resp.Error)
	}

	return nil
}

// ListTasks sends a list tasks command to the daemon
func (c *Client) ListTasks() (interface{}, error) {
	cmd := ipc.NewCommand(ipc.CmdList, nil)

	resp, err := c.SendCommand(cmd)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf(resp.Error)
	}

	return resp.Data, nil
}

// DeleteTask sends a delete task command to the daemon
func (c *Client) DeleteTask(name string) error {
	cmd := ipc.NewCommand(ipc.CmdDelete, map[string]any{
		"name": name,
	})

	resp, err := c.SendCommand(cmd)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf(resp.Error)
	}

	return nil
}

// StopTask sends a stop task command to the daemon
func (c *Client) StopTask(name string) error {
	cmd := ipc.NewCommand(ipc.CmdStop, map[string]any{
		"name": name,
	})

	resp, err := c.SendCommand(cmd)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf(resp.Error)
	}

	return nil
}
