package zai

// ── Request ───────────────────────────────────────────────────────────────────

// Thinking mengontrol mode reasoning GLM-4.7.
// Type: "enabled" | "disabled" (default server-side: "enabled")
type Thinking struct {
	Type string `json:"type"` // "enabled" | "disabled"
}

type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Tools       []Tool    `json:"tools,omitempty"`
	ToolChoice  string    `json:"tool_choice,omitempty"` // "auto" | "none"
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	Thinking    *Thinking `json:"thinking,omitempty"` // GLM-4.7 thinking mode
}

type Message struct {
	Role       string     `json:"role"`                    // "system"|"user"|"assistant"|"tool"
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`    // diisi saat role=assistant + minta tool
	ToolCallID string     `json:"tool_call_id,omitempty"`  // diisi saat role=tool
	Name       string     `json:"name,omitempty"`
}

// ── Tool / Function Definition ─────────────────────────────────────────────────

type Tool struct {
	Type     string   `json:"type"` // selalu "function"
	Function Function `json:"function"`
}

type Function struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"` // JSON Schema
}

// ── Response ──────────────────────────────────────────────────────────────────

type ChatResponse struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`
	Model   string    `json:"model"`
	Choices []Choice  `json:"choices"`
	Usage   Usage     `json:"usage"`
	Error   *APIError `json:"error,omitempty"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"` // "stop" | "tool_calls" | "length"
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // "function"
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string dari GLM
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type APIError struct {
	Code    interface{} `json:"code"`
	Message string      `json:"message"`
}
