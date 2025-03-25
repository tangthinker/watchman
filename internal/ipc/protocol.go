package ipc

import (
	"encoding/json"
	"fmt"
)

// Command represents the command type
type CommandType string

const (
	CmdAdd    CommandType = "ADD"
	CmdList   CommandType = "LIST"
	CmdDelete CommandType = "DELETE"
	CmdStop   CommandType = "STOP"
)

// Command represents a command sent from CLI to daemon
type Command struct {
	Type    CommandType    `json:"type"`
	Payload map[string]any `json:"payload,omitempty"`
}

// Response represents a response sent from daemon to CLI
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Socket path for Unix domain socket
const SockAddr = "/tmp/watchman.sock"

// NewCommand creates a new command with the given type and payload
func NewCommand(cmdType CommandType, payload map[string]any) *Command {
	return &Command{
		Type:    cmdType,
		Payload: payload,
	}
}

// NewResponse creates a new response
func NewResponse(success bool, data interface{}, err error) *Response {
	resp := &Response{
		Success: success,
		Data:    data,
	}
	if err != nil {
		resp.Error = err.Error()
	}
	return resp
}

// Marshal converts Command to JSON bytes
func (c *Command) Marshal() ([]byte, error) {
	return json.Marshal(c)
}

// Unmarshal converts JSON bytes to Command
func UnmarshalCommand(data []byte) (*Command, error) {
	var cmd Command
	if err := json.Unmarshal(data, &cmd); err != nil {
		return nil, fmt.Errorf("failed to unmarshal command: %v", err)
	}
	return &cmd, nil
}

// Marshal converts Response to JSON bytes
func (r *Response) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// Unmarshal converts JSON bytes to Response
func UnmarshalResponse(data []byte) (*Response, error) {
	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	return &resp, nil
}
