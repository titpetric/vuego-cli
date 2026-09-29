package server

// RenderRequest contains template and data for rendering.
type RenderRequest struct {
	Template string            `json:"template"`
	Data     string            `json:"data"`
	Files    map[string]string `json:"files,omitempty"`
}
