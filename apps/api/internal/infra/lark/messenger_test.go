package lark

import (
	"encoding/json"
	"testing"
)

func TestBuildTextContent(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "simple text",
			text: "hello world",
			want: "hello world",
		},
		{
			name: "text with special chars",
			text: `say "hi" to me`,
			want: `say "hi" to me`,
		},
		{
			name: "empty text",
			text: "",
			want: "",
		},
		{
			name: "text with newline",
			text: "line1\nline2",
			want: "line1\nline2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := buildTextContent(tt.text)
			// 验证 JSON 格式正确，并且 text 字段值匹配
			var parsed struct {
				Text string `json:"text"`
			}
			if err := json.Unmarshal([]byte(content), &parsed); err != nil {
				t.Fatalf("content is not valid JSON: %v, content=%s", err, content)
			}
			if parsed.Text != tt.want {
				t.Errorf("text field = %q, want %q", parsed.Text, tt.want)
			}
		})
	}
}

func TestBuildCardContent(t *testing.T) {
	tests := []struct {
		name     string
		cardJSON string
	}{
		{
			name:     "simple card",
			cardJSON: `{"config":{"wide_screen_mode":true},"header":{"title":{"tag":"plain_text","content":"Test"}}}`,
		},
		{
			name:     "empty card",
			cardJSON: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := buildCardContent(tt.cardJSON)
			if content != tt.cardJSON {
				t.Errorf("buildCardContent() = %q, want %q", content, tt.cardJSON)
			}
		})
	}
}
