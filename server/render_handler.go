package server

import (
	"encoding/json"
	"io/fs"
	"net/http"

	"github.com/titpetric/vuego"
)

// RenderHandler returns an http.HandlerFunc that renders templates via POST /render.
// The optional baseFS provides additional files (like components) available during rendering.
func RenderHandler(baseFS fs.FS, opts ...vuego.LoadOption) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost {
			_ = json.NewEncoder(w).Encode(RenderResponse{
				Error: "method not allowed",
			})
			return
		}

		var req RenderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			_ = json.NewEncoder(w).Encode(RenderResponse{
				Error: "invalid JSON: " + err.Error(),
			})
			return
		}

		html, err := Render(r.Context(), baseFS, req, opts...)
		if err != nil {
			_ = json.NewEncoder(w).Encode(RenderResponse{
				Error: err.Error(),
			})
			return
		}

		_ = json.NewEncoder(w).Encode(RenderResponse{
			HTML: html,
		})
	}
}
