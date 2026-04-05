package mcp

import (
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// NewServer creates the MCP server instance.
// The server has no tools registered yet; tool registration is handled
// by a separate codegen pass (see future TypeSpec→MCP emitter work).
func NewServer() *mcpserver.MCPServer {
	return mcpserver.NewMCPServer(
		"Tend",
		"0.1.0",
	)
}
