package main

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
)

type mcpRequestMsg struct {
	apply func(*model) (any, error)
	reply chan<- mcpResponse
}

type mcpResponse struct {
	result any
	err    error
}

type mcpUIBridge struct {
	program *tea.Program
}

func newMCPUIBridge(program *tea.Program) *mcpUIBridge {
	return &mcpUIBridge{program: program}
}

func (b *mcpUIBridge) apply(ctx context.Context, fn func(*model) (any, error)) (any, error) {
	if b == nil || b.program == nil {
		return nil, fmt.Errorf("MCP bridge is not initialized")
	}

	responseCh := make(chan mcpResponse, 1)
	b.program.Send(mcpRequestMsg{apply: fn, reply: responseCh})

	select {
	case response := <-responseCh:
		return response.result, response.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
