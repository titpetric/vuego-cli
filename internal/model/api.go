package model

// EvalRequest is the JSON request body for POST /api/codeblock/eval.
//
// Language selects the evaluator (for example "php", "vuego", "exec", "sql").
// Entry optionally overrides the default entry filename for the language.
// Files maps filename to content; for a single code fence this is usually a
// single entry, while @file references contribute additional companion files.
type EvalRequest struct {
	Language string            `json:"language"`
	Entry    string            `json:"entry,omitempty"`
	Files    map[string]string `json:"files"`
}

// EvalResponse is the JSON response body for POST /api/codeblock/eval.
type EvalResponse struct {
	ContentType string `json:"contentType,omitempty"`
	Content     string `json:"content,omitempty"`
	Error       string `json:"error,omitempty"`
}
