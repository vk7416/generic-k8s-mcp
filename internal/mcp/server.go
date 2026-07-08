package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
)

// Server is a minimal MCP JSON-RPC server over newline-delimited stdio.
type Server struct {
	Name     string
	Version  string
	Registry Registry
	Logger   *log.Logger
}

// RunStdio runs the MCP server over stdin/stdout.
func (s *Server) RunStdio(in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	encoder := json.NewEncoder(out)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			_ = encoder.Encode(Response{JSONRPC: "2.0", Error: &Error{Code: -32700, Message: err.Error()}})
			continue
		}

		resp, ok := s.handle(req)
		if !ok {
			continue
		}
		if err := encoder.Encode(resp); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (s *Server) handle(req Request) (Response, bool) {
	if req.ID == nil && req.Method == "notifications/initialized" {
		return Response{}, false
	}
	if req.ID == nil {
		return Response{}, false
	}

	resp := Response{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		resp.Result = map[string]any{
			"protocolVersion": "2025-06-18",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    s.Name,
				"version": s.Version,
			},
		}
	case "ping":
		resp.Result = map[string]any{}
	case "tools/list":
		resp.Result = map[string]any{"tools": s.Registry.ListTools()}
	case "tools/call":
		var params CallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			resp.Error = &Error{Code: -32602, Message: err.Error()}
			return resp, true
		}
		result, err := s.Registry.CallTool(params.Name, params.Arguments)
		if err != nil {
			resp.Result = ToolResult{
				IsError: true,
				Content: []Content{{Type: "text", Text: err.Error()}},
			}
			return resp, true
		}
		resp.Result = result
	default:
		resp.Error = &Error{Code: -32601, Message: fmt.Sprintf("method %q not found", req.Method)}
	}
	return resp, true
}
