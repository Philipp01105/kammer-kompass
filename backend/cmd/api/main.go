package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"

	"github.com/Philipp01105/kammer-kompass/backend/internal/admin"
	"github.com/Philipp01105/kammer-kompass/backend/internal/audit"
	"github.com/Philipp01105/kammer-kompass/backend/internal/auth"
	"github.com/Philipp01105/kammer-kompass/backend/internal/bootstrap"
	"github.com/Philipp01105/kammer-kompass/backend/internal/config"
	"github.com/Philipp01105/kammer-kompass/backend/internal/db"
	"github.com/Philipp01105/kammer-kompass/backend/internal/db/sqlc"
	"github.com/Philipp01105/kammer-kompass/backend/internal/httpx"
	appmw "github.com/Philipp01105/kammer-kompass/backend/internal/middleware"
	"github.com/Philipp01105/kammer-kompass/backend/internal/migrations"
	"github.com/Philipp01105/kammer-kompass/backend/internal/moderation"
	"github.com/Philipp01105/kammer-kompass/backend/internal/public"
	"github.com/Philipp01105/kammer-kompass/backend/internal/rate_limit"
	"github.com/Philipp01105/kammer-kompass/backend/internal/rbac"
)

func main() {
	cfg, err := config.FromEnv()
	if err != nil {
		slog.Error("config error", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dbpool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db connect error", "error", err)
		os.Exit(1)
	}
	defer dbpool.Close()

	if err := migrations.Run(ctx, dbpool); err != nil {
		slog.Error("migration error", "error", err)
		os.Exit(1)
	}

	if err := bootstrap.SyncRoleTemplates(ctx, dbpool); err != nil {
		slog.Error("role template sync error", "error", err)
		os.Exit(1)
	}

	if err := bootstrap.SyncIHKCatalog(ctx, dbpool); err != nil {
		slog.Error("ihk catalog sync error", "error", err)
		os.Exit(1)
	}

	credentials, err := bootstrap.EnsureDefaultSuperAdmin(ctx, dbpool)
	if err != nil {
		slog.Error("bootstrap error", "error", err)
		os.Exit(1)
	}
	if credentials != nil {
		slog.Warn("default super_admin created", "username", credentials.Username, "password", credentials.Password)
	}

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer func() { _ = redisClient.Close() }()

	sessionMgr, err := auth.NewSessionManager(cfg.Session)
	if err != nil {
		slog.Error("session init error", "error", err)
		os.Exit(1)
	}

	q := sqlc.New(dbpool)

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.ClientIPFromHeader("X-Forwarded-For"))
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(15 * time.Second))
	r.Use(appmw.CORS([]string{
		"http://localhost:3000",
		"http://127.0.0.1:3000",
	}))
	r.Use(httpx.SlogAccessLog(logger))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	limiter := rate_limit.New(redisClient)

	authHandler, err := auth.NewHandler(dbpool, cfg.Session, limiter, cfg.SecretSalt)
	if err != nil {
		slog.Error("auth init error", "error", err)
		os.Exit(1)
	}

	// auth route mapping
	r.Post("/api/v1/register", authHandler.Register)
	r.Post("/api/v1/login", authHandler.Login)
	r.Post("/api/v1/logout", authHandler.Logout)
	r.Get("/api/v1/me", authHandler.Me)
	r.Get("/api/v1/role-templates", authHandler.ListRequestableRoleTemplates)
	r.With(appmw.RequireAuth(sessionMgr)).Post("/api/v1/permission-requests", authHandler.RequestPermissions)

	// needed public services
	langDetector := moderation.NewLinguaDetector()
	publicHandler := public.NewHandler(q, limiter, cfg.SecretSalt, langDetector)

	// public route mapping
	r.Route("/api/v1/public", func(r chi.Router) {
		r.Use(appmw.OptionalAuth(sessionMgr))
		r.Get("/ihks", publicHandler.ListIHKs)
		r.Get("/ihks/{slug}", publicHandler.GetIHKBySlug)
		r.Post("/info-suggestions", publicHandler.SubmitInfoSuggestion)
	})

	// needed admin services
	rbacSvc := rbac.NewService(q)
	auditWriter := audit.NewWriter(q, cfg.SecretSalt)
	adminHandler := admin.NewHandler(dbpool, q, rbacSvc, auditWriter, cfg.SecretSalt)
	globalScope := func(*http.Request) (rbac.ResourceScope, error) {
		return rbac.ResourceScope{}, nil
	}

	// admin route mapping
	r.Route("/api/v1/admin", func(r chi.Router) {
		r.Use(appmw.RequireAuth(sessionMgr))

		r.Get("/me", adminHandler.Me)

		r.With(appmw.RequirePermissions(rbacSvc, rbac.ActionManageModerationTerms, globalScope)).
			Get("/moderation-terms", adminHandler.ListModerationTerms)
		r.With(appmw.RequirePermissions(rbacSvc, rbac.ActionManageModerationTerms, globalScope)).
			Post("/moderation-terms", adminHandler.CreateModerationTerm)
		r.With(appmw.RequirePermissions(rbacSvc, rbac.ActionManageModerationTerms, globalScope)).
			Patch("/moderation-terms/{id}", adminHandler.UpdateModerationTerm)
		r.With(appmw.RequirePermissions(rbacSvc, rbac.ActionManageModerationTerms, globalScope)).
			Delete("/moderation-terms/{id}", adminHandler.DeleteModerationTerm)

		r.With(appmw.RequirePermissions(rbacSvc, rbac.PermAuditRead, globalScope)).
			Get("/audit-logs", adminHandler.ListAuditLogs)
		r.With(appmw.RequirePermissions(rbacSvc, rbac.PermUserRead, globalScope)).
			Get("/users", adminHandler.ListUsers)
		r.With(appmw.RequirePermissions(rbacSvc, rbac.PermUserRead|rbac.PermUserUpdate|rbac.PermAuditWrite, globalScope)).
			Post("/users", adminHandler.CreateUser)
		r.With(appmw.RequirePermissions(rbacSvc, rbac.PermUserRead, globalScope)).
			Get("/role-templates", adminHandler.ListRoleTemplates)
		r.With(appmw.RequirePermissions(rbacSvc, rbac.PermUserRead, globalScope)).
			Get("/users/{id}/roles", adminHandler.ListUserRoleAssignments)
		r.With(appmw.RequirePermissions(rbacSvc, rbac.ActionAssignRole, globalScope)).
			Post("/users/{id}/roles", adminHandler.AssignUserRole)
		r.With(appmw.RequirePermissions(rbacSvc, rbac.PermUserRead|rbac.PermRoleRevoke|rbac.PermAuditWrite, globalScope)).
			Delete("/users/{id}/roles/{assignmentId}", adminHandler.RevokeUserRole)
		r.With(appmw.RequirePermissions(rbacSvc, rbac.ActionAssignRole, globalScope)).
			Get("/permission-requests", adminHandler.ListPermissionRequests)
		r.With(appmw.RequirePermissions(rbacSvc, rbac.ActionAssignRole, globalScope)).
			Get("/permission-requests/{id}", adminHandler.GetPermissionRequest)
		r.With(appmw.RequirePermissions(rbacSvc, rbac.ActionAssignRole, globalScope)).
			Post("/permission-requests/{id}/approve", adminHandler.ApprovePermissionRequest)
		r.With(appmw.RequirePermissions(rbacSvc, rbac.ActionAssignRole, globalScope)).
			Post("/permission-requests/{id}/reject", adminHandler.RejectPermissionRequest)

		r.Get("/ihks", adminHandler.ListIHKs)
		r.Patch("/ihks/{id}", adminHandler.UpdateIHK)
		r.Get("/ihks/{id}/info/versions", adminHandler.ListIHKInfoVersions)
		r.Post("/ihks/{id}/info/publish", adminHandler.PublishIHKInfo)
		r.Post("/ihks/{id}/info/rollback", adminHandler.RollbackIHKInfo)

		r.Get("/info-suggestions", adminHandler.ListInfoSuggestions)
		r.Get("/info-suggestions/{id}", adminHandler.GetInfoSuggestion)
		r.Post("/info-suggestions/{id}/start-review", adminHandler.StartReviewInfoSuggestion)
		r.Post("/info-suggestions/{id}/accept", adminHandler.AcceptInfoSuggestion)
		r.Post("/info-suggestions/{id}/reject", adminHandler.RejectInfoSuggestion)
		r.Post("/info-suggestions/{id}/needs-more-info", adminHandler.NeedsMoreInfoSuggestion)
		r.Post("/info-suggestions/{id}/mark-spam", adminHandler.MarkSpamInfoSuggestion)
		r.Post("/info-suggestions/{id}/reopen", adminHandler.ReopenInfoSuggestion)
		r.Post("/info-suggestions/{id}/hide-pending", adminHandler.HidePendingInfoSuggestion)
		r.Post("/info-suggestions/{id}/apply", adminHandler.ApplyInfoSuggestion)

	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("http server listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server error", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
