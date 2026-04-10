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
