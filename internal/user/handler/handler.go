package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"

	"authe/internal/user"
)

type Service interface {
	Create(context.Context, *user.User) (int, error)
	Get(context.Context, int) (*user.User, error)
	Update(context.Context, int, *user.User) error
	Delete(context.Context, int) error
	IsExistByUsername(context.Context, string) error
	Login(context.Context, string, string) (string, error)
	Logout(ctx context.Context, token string) error
}
type userHandler struct {
	Service Service
}

func NewHandler(service Service) *userHandler {
	return &userHandler{Service: service}
}

func (h *userHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "failed to decode user", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		http.Error(w, "all fields are required", http.StatusBadRequest)
		return
	}

	id, err := h.Service.Create(r.Context(), req.ToDomain())
	if err != nil {
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "user created",
		"id": strconv.Itoa(id)})
}

func (h *userHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	user, err := h.Service.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "failed to get user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ToUserResponse(user))
}

func (h *userHandler) Login(w http.ResponseWriter, r *http.Request) {
	var u user.User

	json.NewDecoder(r.Body).Decode(&u)

	token, err := h.Service.Login(r.Context(), u.Username, u.Password)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, fmt.Sprintf("user not found: %s", err), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("unknown server error: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func (h *userHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if token == "" {
		http.Error(w, "no token provided", http.StatusUnauthorized)
		return
	}

	err := h.Service.Logout(r.Context(), token)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to logout: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "user logged out"})
}

func (h *userHandler) Welcome(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value("username").(string)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"username": username})

}
func (h *userHandler) Update(w http.ResponseWriter, r *http.Request) {

}

func (h *userHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	err = h.Service.Delete(r.Context(), id)
	if err != nil {
		http.Error(w, "failed to delete user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "user deleted"})
}
