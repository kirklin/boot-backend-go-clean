package route

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/gateway"
	"github.com/kirklin/boot-backend-go-clean/internal/infrastructure/persistence"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/controller"
	"github.com/kirklin/boot-backend-go-clean/internal/usecase"
	"github.com/kirklin/boot-backend-go-clean/pkg/configs"
	"github.com/kirklin/boot-backend-go-clean/pkg/database"
)

func NewAuthRouter(db database.Database, api huma.API, config *configs.AppConfig, authenticator gateway.Authenticator) {
	ur := persistence.NewUserRepository(db)
	txm := persistence.NewTxManager(db)
	ac := controller.NewAuthController(usecase.NewAuthUseCase(ur, authenticator, txm, config))

	// Public auth routes (no middleware needed — huma handles request binding)
	huma.Register(api, huma.Operation{
		OperationID: "auth-register",
		Method:      http.MethodPost,
		Path:        "/v1/api/auth/register",
		Summary:     "Register a new user",
		Description: "Create a new user account with username, email and password.",
		Tags:        []string{"Auth"},
	}, ac.Register)

	huma.Register(api, huma.Operation{
		OperationID: "auth-login",
		Method:      http.MethodPost,
		Path:        "/v1/api/auth/login",
		Summary:     "User login",
		Description: "Authenticate with username and password to receive JWT tokens.",
		Tags:        []string{"Auth"},
	}, ac.Login)

	huma.Register(api, huma.Operation{
		OperationID: "auth-refresh",
		Method:      http.MethodPost,
		Path:        "/v1/api/auth/refresh",
		Summary:     "Refresh access token",
		Description: "Exchange a valid refresh token for a new access/refresh token pair.",
		Tags:        []string{"Auth"},
	}, ac.RefreshToken)

	huma.Register(api, huma.Operation{
		OperationID: "auth-logout",
		Method:      http.MethodPost,
		Path:        "/v1/api/auth/logout",
		Summary:     "User logout",
		Description: "Revoke the refresh token to log out the user.",
		Tags:        []string{"Auth"},
		Security: []map[string][]string{
			{"bearerAuth": {}},
		},
	}, ac.Logout)
}
