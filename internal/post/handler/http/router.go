package http

import (
	"net/http"
)

func (h *PostHandler) Route(mux *http.ServeMux) {
	mux.HandleFunc("POST /posts", h.CreatePost)
}
