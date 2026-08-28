package http

import (
	"encoding/json"
	"net/http"

	userV1 "authe/pkg/proto/user/v1"
)

type PostHandler struct {
	userClient userV1.UserServiceClient
}

func NewPostHandler(userClient userV1.UserServiceClient) *PostHandler {
	return &PostHandler{
		userClient: userClient,
	}
}

type CreatePostRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type CreatePostResponse struct {
	ID      int64  `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Author  string `json:"author"`
}

func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Invalid Authorization", http.StatusUnauthorized)
		return
	}

	var req userV1.VerifyTokenRequest
	req.Token = authHeader

	resp, err := h.userClient.VerifyToken(r.Context(), &req)
	if err != nil || !resp.GetValid() {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req2 CreatePostRequest
	err = json.NewDecoder(r.Body).Decode(&req2)
	if err != nil {
		http.Error(w, "Invalid Request", http.StatusBadRequest)
		return
	}

	post := CreatePostResponse{
		ID:      1,
		Title:   req2.Title,
		Content: req2.Content,
		Author:  resp.GetUsername(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(post)
}
