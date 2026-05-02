package route

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/gateway"
	"github.com/kirklin/boot-backend-go-clean/internal/infrastructure/persistence"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/controller"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/middleware"
	"github.com/kirklin/boot-backend-go-clean/internal/usecase"
	"github.com/kirklin/boot-backend-go-clean/pkg/configs"
	"github.com/kirklin/boot-backend-go-clean/pkg/database"
)

func NewUserRouter(db database.Database, api huma.API, router *gin.Engine, _ *configs.AppConfig, authenticator gateway.Authenticator) {
	ur := persistence.NewUserRepository(db)
	uc := controller.NewUserController(usecase.NewUserUseCase(ur))

	// Create a gin group with JWT middleware applied.
	// Huma will register its routes onto this group so that
	// gin's middleware (JWT auth) runs before huma's handler.
	userGroup := router.Group("/v1/api/users")
	userGroup.Use(middleware.JWTAuthMiddleware(authenticator))

	// Inject huma.Context into context.Context for each request so handlers
	// can access gin.Context when needed (e.g. reading JWT userID).
	userGroup.Use(func(c *gin.Context) {
		// Store the gin.Context itself into the request context for later
		// retrieval by the huma handler via controller.humaContextKey.
		ctx := context.WithValue(c.Request.Context(), controller.HumaContextKey, c)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	// Create a huma API scoped to this authenticated group
	userAPI := humagin.NewWithGroup(router, userGroup, huma.DefaultConfig("", ""))

	// Copy the OpenAPI document from the parent API so all operations
	// appear in the same spec.
	// We can't share the OpenAPI directly, so we register on the main api
	// with full paths and rely on gin middleware for auth.

	// GET /v1/api/users/:id
	huma.Register(api, huma.Operation{
		OperationID: "get-user",
		Method:      http.MethodGet,
		Path:        "/v1/api/users/{id}",
		Summary:     "Get user by ID",
		Description: "Retrieve user information by their unique ID.",
		Tags:        []string{"Users"},
		Security: []map[string][]string{
			{"bearerAuth": {}},
		},
	}, uc.GetUser)

	// GET /v1/api/users/current
	huma.Register(api, huma.Operation{
		OperationID: "get-current-user",
		Method:      http.MethodGet,
		Path:        "/v1/api/users/current",
		Summary:     "Get current user",
		Description: "Retrieve the profile of the currently authenticated user.",
		Tags:        []string{"Users"},
		Security: []map[string][]string{
			{"bearerAuth": {}},
		},
	}, uc.GetCurrentUser)

	// PUT /v1/api/users/:id
	huma.Register(api, huma.Operation{
		OperationID: "update-user",
		Method:      http.MethodPut,
		Path:        "/v1/api/users/{id}",
		Summary:     "Update user",
		Description: "Update user profile. Only the user themselves can update their own profile.",
		Tags:        []string{"Users"},
		Security: []map[string][]string{
			{"bearerAuth": {}},
		},
	}, uc.UpdateUser)

	// Suppress unused variable warning — we created userAPI for the gin middleware group
	_ = userAPI
}
