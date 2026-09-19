package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"haircutz/backend/internal/auth"
	"haircutz/backend/internal/model"
	"haircutz/backend/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrNotAdminRole       = errors.New("admin access required")
)

type AuthController struct {
	admins *repository.AdminRepository
	tokens *auth.TokenIssuer
}

func NewAuthController(admins *repository.AdminRepository, tokens *auth.TokenIssuer) *AuthController {
	return &AuthController{admins: admins, tokens: tokens}
}

type LoginResult struct {
	Token     string
	ExpiresAt string
	Role      model.Role
	Email     string
}

func (c *AuthController) Login(ctx context.Context, email, password string) (LoginResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || password == "" {
		return LoginResult{}, ErrInvalidCredentials
	}

	admin, err := c.admins.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrAdminNotFound) {
			return LoginResult{}, ErrInvalidCredentials
		}
		return LoginResult{}, fmt.Errorf("login lookup: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}

	if admin.Role != model.RoleAdmin {
		return LoginResult{}, ErrNotAdminRole
	}

	token, expiresAt, err := c.tokens.Issue(admin)
	if err != nil {
		return LoginResult{}, fmt.Errorf("issue token: %w", err)
	}

	return LoginResult{
		Token:     token,
		ExpiresAt: expiresAt.Format("2006-01-02T15:04:05Z07:00"),
		Role:      admin.Role,
		Email:     admin.Email,
	}, nil
}
