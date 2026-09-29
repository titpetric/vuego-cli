package model

// EvalResponse is the JSON response body for POST /api/codeblock/eval.
type EvalResponse struct {
	ContentType string `json:"contentType,omitempty"`
	Content     string `json:"content,omitempty"`
	Error       string `json:"error,omitempty"`
}
