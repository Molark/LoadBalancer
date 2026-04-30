package handlers

import (
	"booking/internal/contextKeys"
	"booking/internal/models"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var adminId = "00000000-0000-0000-0000-000000000001"
var userId = "00000000-0000-0000-0000-000000000002"

type CustomClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrInternal     = errors.New("internal error")
)

func (h Handler) dummyLogin(w http.ResponseWriter, r *http.Request) {
	type Role struct {
		Role string `json:"role"`
	}
	role := Role{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&role)
	if err != nil {
		RespondError(w, ErrInvalidRequest)
		slog.Error("Error in  parsing dummy Login", slog.Any("err", err))
	}
	if !(role.Role == "admin" || role.Role == "user") {
		RespondError(w, ErrInvalidRequest)
		slog.Error("Invalid role", slog.Any("role", role))
		return
	}
	var id string
	if role.Role == "admin" {
		id = adminId
	} else if role.Role == "user" {
		id = userId
	}
	token, err := h.auth.CreateToken(id, role.Role)
	if err != nil {
		RespondError(w, ErrInternal)
		slog.Error("Error creating token", slog.Any("err", err))
		return
	}
	type Token struct {
		Token string `json:"token"`
	}
	RespondJSON(w, http.StatusOK, Token{Token: token})

}

func (h Handler) JWTAuthMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(auth, prefix) {
			RespondError(w, ErrUnauthorized)
			return
		}

		token := strings.TrimPrefix(auth, prefix)
		claims, err := h.auth.ParseToken(token)
		if err != nil {
			RespondError(w, ErrUnauthorized)
			return
		}

		id, err := uuid.Parse(claims.Subject)
		if err != nil {
			RespondError(w, ErrInternal)
			return
		}
		ctx := context.WithValue(r.Context(), contextKeys.UserIdKey, id)
		ctx = context.WithValue(ctx, contextKeys.RoleKey, claims.Role)
		next.ServeHTTP(w, r.WithContext(ctx))

	})
}
func (h Handler) Register(w http.ResponseWriter, r *http.Request) {
	type RegisterRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}

	var req RegisterRequest
	err := ReadRequestBody(r, &req)
	if err != nil {
		RespondError(w, ErrInvalidRequest)
		slog.Error("Error in parsing Register", slog.Any("err", err))
		return
	}

	if req.Email == "" || req.Password == "" || !(req.Role == "admin" || req.Role == "user") {
		RespondError(w, ErrInvalidRequest)
		slog.Error("Invalid register request data", slog.Any("role", req.Role))
		return
	}

	hashedPassword, err := h.auth.HashPassword(req.Password)
	if err != nil {
		RespondError(w, ErrInternal)
		slog.Error("Error hashing password for register", slog.Any("err", err))
		return
	}

	userToCreate := models.User{
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Role:         req.Role,
	}

	created, err := h.service.CreateUser(r.Context(), userToCreate)
	if err != nil {
		RespondError(w, ErrInvalidRequest)
		slog.Error("Error creating user in register", slog.Any("err", err))
		return
	}

	type Response struct {
		User models.User `json:"user"`
	}
	RespondJSON(w, http.StatusCreated, Response{User: created})
}

func (h Handler) Login(w http.ResponseWriter, r *http.Request) {
	type LoginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var req LoginRequest
	err := ReadRequestBody(r, &req)
	if err != nil {
		RespondError(w, ErrInvalidRequest)
		slog.Error("Error in parsing Login", slog.Any("err", err))
		return
	}

	if req.Email == "" || req.Password == "" {
		RespondError(w, ErrInvalidRequest)
		slog.Error("Invalid login request data")
		return
	}

	user, err := h.service.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			RespondError(w, ErrUnauthorized)
		} else {
			RespondError(w, ErrInternal)
			slog.Error("Error in GetUserByEmail for login", slog.Any("err", err))
		}
		return
	}

	if !h.auth.VerifyPassword(req.Password, user.PasswordHash) {
		RespondError(w, ErrUnauthorized)
		return
	}

	token, err := h.auth.CreateToken(user.Id.String(), user.Role)
	if err != nil {
		RespondError(w, ErrInternal)
		slog.Error("Error creating token in login", slog.Any("err", err))
		return
	}

	type Token struct {
		Token string `json:"token"`
	}
	RespondJSON(w, http.StatusOK, Token{Token: token})
}
