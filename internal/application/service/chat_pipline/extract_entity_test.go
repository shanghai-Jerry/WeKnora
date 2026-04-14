package chatpipline

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
)

func TestExtractor_ExtractStream(t *testing.T) {
	templateExample := []types.GraphData{}

	promptBytes, _ := os.ReadFile("/Users/moineye/workspace/gopath/src/github.com/shanghai-Jerry/WeKnora/python-scripts/prompt/extract_entity_prompt.txt")
	templateDescription := strings.TrimSpace(string(promptBytes))
	// Test case: Successful extraction
	template := &types.PromptTemplateStructured{
		Description: templateDescription,
		Examples:    templateExample,
	}

	ctx := context.Background()

	os.Setenv("OLLAMA_BASE_URL", "http://localhost:11434")

	model, err := chat.NewRemoteChat(&chat.ChatConfig{
		ModelName: "deepseek-chat",
		APIKey:    "",
		BaseURL:   "https://api.deepseek.com/v1",
		Provider:  "deepseek",
	})
	if err != nil {
		t.Errorf("Failed to create Ollama chat: %v", err)
	}
	query := "一名患有2型糖尿病15年且合并高度近视的患者，近期出现视力急剧下降和飞蚊症增多，应考虑哪些可能的并发症？确诊需要做哪些检查？不同分期对应的治疗方案有何区别"
	extractor := NewExtractor(model, template)
	graph, err := extractor.Extract(ctx, query)
	// t.Logf("system:\n%v", generator.System(ctx))
	// t.Logf("user:%v\n", generator.User(ctx, query))
	if err != nil {
		t.Errorf("Failed to extract entities: %v", err)
	}
	nodes := []string{}
	for _, node := range graph.Node {
		nodes = append(nodes, node.Name)
	}
	t.Logf("Extracted nodes: %v", nodes)
}

func TestFormater_extractContent(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		textPath string
		want     string
	}{
		{
			name:     "test",
			textPath: "/Users/moineye/workspace/gopath/src/github.com/shanghai-Jerry/WeKnora/internal/application/service/chat_pipline/extract_data/1.txt",
			want:     "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFormater()
			textBytes, _ := os.ReadFile(tt.textPath)
			got := f.extractContent(context.Background(), string(textBytes))
			// TODO: update the condition below to compare got with tt.want.
			t.Logf("got:%v", got)
		})
	}
}

func TestFormater_parseOutput(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		text    string
		want    []map[string]interface{}
		wantErr bool
	}{
		{
			name:     "test",
			text: "/Users/moineye/workspace/gopath/src/github.com/shanghai-Jerry/WeKnora/internal/application/service/chat_pipline/extract_data/1.txt",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFormater()
			textBytes, _ := os.ReadFile(tt.text)
			tt.text = string(textBytes)
			got, gotErr := f.parseOutput(context.Background(), tt.text)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("parseOutput() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("parseOutput() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("parseOutput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormater_ParseGraph(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		text    string
		want    *types.GraphData
		wantErr bool
	}{
		{
			name:     "test",
			text: "/Users/moineye/workspace/gopath/src/github.com/shanghai-Jerry/WeKnora/internal/application/service/chat_pipline/extract_data/1.txt",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFormater()
			textBytes, _ := os.ReadFile(tt.text)
			tt.text = string(textBytes)
			got, gotErr := f.ParseGraph(context.Background(), tt.text)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("ParseGraph() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("ParseGraph() succeeded unexpectedly")
			}
			
			var names []string
			for _, node := range got.Node {
				names = append(names, node.Name)
			}
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("ParseGraph() = %v, want %v", names, tt.want)
			}
		})
	}
}
