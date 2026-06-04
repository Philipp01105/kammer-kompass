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
	"github.com/Philipp01105/kammer-kompass/backend/internal/netx"
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

	if cfg.BootstrapSuperAdmin {
		credentials, err := bootstrap.EnsureDefaultSuperAdmin(ctx, dbpool, cfg.BootstrapPassword)
		if err != nil {
			slog.Error("bootstrap error", "error", err)
			os.Exit(1)
		}
		if credentials != nil {
			slog.Warn("bootstrap super_admin created; rotate the configured bootstrap password immediately", "username", credentials.Username)
		}
	} else {
		slog.Debug("super_admin bootstrap skipped (BOOTSTRAP_SUPER_ADMIN not set)")
	}

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer func() { _ = redisClient.Close() }()

	sessionMgr, err := auth.NewSessionManager(cfg.Session, redisClient)
	if err != nil {
		slog.Error("session init error", "error", err)
		os.Exit(1)
	}

	q := sqlc.New(dbpool)

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(appmw.RealIP(netx.ParseCIDRs(cfg.TrustedProxies)))
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(15 * time.Second))
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB
			next.ServeHTTP(w, r)
		})
	})
	r.Use(appmw.CORS(cfg.AllowedOrigins))
	r.Use(httpx.SlogAccessLog(logger))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	limiter := rate_limit.New(redisClient)

	authHandler, err := auth.NewHandler(dbpool, sessionMgr, limiter, cfg.SecretSalt)
	if err != nil {
		slog.Error("auth init error", "error", err)
		os.Exit(1)
	}

	// auth route mapping
	r.Post("/api/v1/login", authHandler.Login)
	r.Post("/api/v1/logout", authHandler.Logout)
	r.Get("/api/v1/me", authHandler.Me)

	// needed public services
	langDetector := moderation.NewLinguaDetector()
	publicHandler, err := public.NewHandler(q, limiter, cfg.SecretSalt, langDetector)
	if err != nil {
		slog.Error("public handler init error", "error", err)
		os.Exit(1)
	}

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
	adminHandler := admin.NewHandler(dbpool, q, rbacSvc, auditWriter, sessionMgr, cfg.SecretSalt)
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
		r.With(appmw.RequirePermissions(rbacSvc, rbac.PermSystemAdmin|rbac.PermUserRead|rbac.PermUserUpdate|rbac.PermAuditWrite, globalScope)).
			Post("/users", adminHandler.CreateUser)
		r.With(appmw.RequirePermissions(rbacSvc, rbac.PermUserUpdate|rbac.PermAuditWrite, globalScope)).
			Patch("/users/{id}/status", adminHandler.UpdateUserStatus)
		r.With(appmw.RequirePermissions(rbacSvc, rbac.PermUserRead, globalScope)).
			Get("/role-templates", adminHandler.ListRoleTemplates)
		r.With(appmw.RequirePermissions(rbacSvc, rbac.PermUserRead, globalScope)).
			Get("/users/{id}/roles", adminHandler.ListUserRoleAssignments)
		r.With(appmw.RequirePermissions(rbacSvc, rbac.ActionAssignRole, globalScope)).
			Post("/users/{id}/roles", adminHandler.AssignUserRole)
		r.With(appmw.RequirePermissions(rbacSvc, rbac.PermUserRead|rbac.PermRoleRevoke|rbac.PermAuditWrite, globalScope)).
			Delete("/users/{id}/roles/{assignmentId}", adminHandler.RevokeUserRole)
		// Scoped routes get a coarse route-level permission gate here. Handlers
		// still resolve the exact IHK/state scope from the DB and perform the
		// final EffectiveMask check before touching the resource.
		r.With(appmw.RequireAnyAssignmentPermissions(rbacSvc, rbac.PermIHKRead)).
			Get("/ihks", adminHandler.ListIHKs)
		r.With(appmw.RequireAnyAssignmentPermissions(rbacSvc, rbac.PermIHKUpdate|rbac.PermAuditWrite)).
			Patch("/ihks/{id}", adminHandler.UpdateIHK)
		r.With(appmw.RequireAnyAssignmentPermissions(rbacSvc, rbac.PermVersionRead)).
			Get("/ihks/{id}/info/versions", adminHandler.ListIHKInfoVersions)
		r.With(appmw.RequireAnyAssignmentPermissions(rbacSvc, rbac.PermIHKRead|rbac.PermInfoPublish|rbac.PermVersionCreate|rbac.PermAuditWrite)).
			Post("/ihks/{id}/info/publish", adminHandler.PublishIHKInfo)
		r.With(appmw.RequireAnyAssignmentPermissions(rbacSvc, rbac.PermInfoRollback|rbac.PermVersionRead|rbac.PermVersionCreate|rbac.PermAuditWrite)).
			Post("/ihks/{id}/info/rollback", adminHandler.RollbackIHKInfo)

		r.With(appmw.RequireAnyAssignmentPermissions(rbacSvc, rbac.PermInfoSuggestionRead)).
			Get("/info-suggestions", adminHandler.ListInfoSuggestions)
		r.With(appmw.RequireAnyAssignmentPermissions(rbacSvc, rbac.PermInfoSuggestionRead)).
			Get("/info-suggestions/{id}", adminHandler.GetInfoSuggestion)
		r.With(appmw.RequireAnyAssignmentPermissions(rbacSvc, rbac.ActionReviewInfoSuggestion)).
			Post("/info-suggestions/{id}/start-review", adminHandler.StartReviewInfoSuggestion)
		r.With(appmw.RequireAnyAssignmentPermissions(rbacSvc, rbac.ActionAcceptInfoSuggestion)).
			Post("/info-suggestions/{id}/accept", adminHandler.AcceptInfoSuggestion)
		r.With(appmw.RequireAnyAssignmentPermissions(rbacSvc, rbac.ActionRejectInfoSuggestion)).
			Post("/info-suggestions/{id}/reject", adminHandler.RejectInfoSuggestion)
		r.With(appmw.RequireAnyAssignmentPermissions(rbacSvc, rbac.ActionReviewInfoSuggestion)).
			Post("/info-suggestions/{id}/needs-more-info", adminHandler.NeedsMoreInfoSuggestion)
		r.With(appmw.RequireAnyAssignmentPermissions(rbacSvc, rbac.PermInfoSuggestionRead|rbac.PermSpamModerate|rbac.PermAuditWrite)).
			Post("/info-suggestions/{id}/mark-spam", adminHandler.MarkSpamInfoSuggestion)
		r.With(appmw.RequireAnyAssignmentPermissions(rbacSvc, rbac.ActionReviewInfoSuggestion)).
			Post("/info-suggestions/{id}/reopen", adminHandler.ReopenInfoSuggestion)
		r.With(appmw.RequireAnyAssignmentPermissions(rbacSvc, rbac.ActionHidePendingHint)).
			Post("/info-suggestions/{id}/hide-pending", adminHandler.HidePendingInfoSuggestion)
		r.With(appmw.RequireAnyAssignmentPermissions(rbacSvc, rbac.ActionApplyInfoSuggestion)).
			Post("/info-suggestions/{id}/apply", adminHandler.ApplyInfoSuggestion)

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
