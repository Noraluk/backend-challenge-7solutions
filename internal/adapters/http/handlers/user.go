package handlers

import (
	"net/http"

	httpdto "github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/dto"
	httperrors "github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/errors"
	"github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/httpx"
	applicationdto "github.com/Noraluk/backend-challenge-7solutions/internal/application/dto"
	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
)

type UserHandler struct {
	users ports.UserUseCase
}

func NewUserHandler(users ports.UserUseCase) *UserHandler {
	return &UserHandler{users: users}
}

func (handler *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	user, err := handler.users.GetUser(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteApplicationError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, user)
}

func (handler *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := handler.users.ListUsers(r.Context())
	if err != nil {
		httpx.WriteApplicationError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, users)
}

func (handler *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	var request httpdto.UpdateUserRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httperrors.CodeInvalidRequest, httperrors.MessageInvalidRequest)
		return
	}

	user, err := handler.users.UpdateUser(r.Context(), applicationdto.UpdateUserInput{
		ID: r.PathValue("id"), Name: request.Name, Email: request.Email,
	})
	if err != nil {
		httpx.WriteApplicationError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, user)
}

func (handler *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	if err := handler.users.DeleteUser(r.Context(), r.PathValue("id")); err != nil {
		httpx.WriteApplicationError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
