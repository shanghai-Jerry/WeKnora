package tools_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Tencent/WeKnora/internal/agent/tools"
	"github.com/Tencent/WeKnora/internal/sandbox"
	"github.com/Tencent/WeKnora/internal/types"
)

func TestCodeInterpreterTool_Execute(t *testing.T) {
	image := "wechatopenai/weknora-sandbox:dev-1.0"
	sandboxMgr, err := sandbox.NewManagerFromType("docker", true, image)

	if err != nil {
		t.Fatalf("Failed to create Docker sandbox: %v", err)
	}
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		sandboxMgr sandbox.Manager
		sessionID  string
		// Named input parameters for target function.
		args    json.RawMessage
		want    *types.ToolResult
		wantErr bool
	}{
		{
			name:       "docker-sandbox",
			sandboxMgr: sandboxMgr,
			sessionID:  "123",
			args:       json.RawMessage(`{"code": "print('Hello, World!')", "language": "python"}`),
			want: &types.ToolResult{
				Success: true,
				Output:  "Hello, World!",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tools.NewCodeInterpreterTool(tt.sandboxMgr, tt.sessionID)
			got, gotErr := c.Execute(context.Background(), tt.args)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Execute() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Execute() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}
