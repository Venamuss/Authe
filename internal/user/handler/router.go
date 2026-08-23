package user

import (
	"net/http"

	"github.com/justinas/alice"
)

func (h *userHandler) Route(mux *http.ServeMux) {

	standart := alice.New(Logger, Recover)
	secured := alice.New(Logger, Recover, Protect)

	mux.Handle("POST /user/create", standart.ThenFunc(h.Create))
	mux.Handle("GET /user/{id}", standart.ThenFunc(h.Get))
	mux.Handle("PATCH /user/update/{id}", standart.ThenFunc(h.Update))
	mux.Handle("DELETE /user/delete/{id}", standart.ThenFunc(h.Delete))
	mux.Handle("POST /user/login", standart.ThenFunc(h.Login))
	mux.Handle("GET /user/welcome", secured.ThenFunc(h.Welcome))
}
