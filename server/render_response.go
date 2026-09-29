package server

// RenderResponse contains the rendered HTML or an error.
type RenderResponse struct {
	HTML  string `json:"html,omitempty"`
	Error string `json:"error,omitempty"`
}
