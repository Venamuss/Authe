package user

import (
	"net/http"

	"github.com/justinas/alice"
)

func (h *userHandler) Route(mux *http.ServeMux, middleware *Middleware) {

	standart := alice.New(middleware.Logger, middleware.Recover)
	secured := alice.New(middleware.Logger, middleware.Recover, middleware.Protect)
	rateLimit := alice.New(middleware.Logger, middleware.Recover, middleware.RateLimit)

	mux.Handle("POST /user/create", rateLimit.ThenFunc(h.Create))
	mux.Handle("GET /user/{id}", standart.ThenFunc(h.Get))
	mux.Handle("PATCH /user/update/{id}", standart.ThenFunc(h.Update))
	mux.Handle("DELETE /user/delete/{id}", standart.ThenFunc(h.Delete))
	mux.Handle("POST /user/login", rateLimit.ThenFunc(h.Login))
	mux.Handle("GET /user/welcome", secured.ThenFunc(h.Welcome))
	mux.Handle("POST /user/logout", secured.ThenFunc(h.Logout))
}
