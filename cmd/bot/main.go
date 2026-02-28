package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"go.uber.org/zap"

	_ "modernc.org/sqlite"

	"mikrotik-mcp/internal/ai/bridge"
	"mikrotik-mcp/internal/ai/zai"
	"mikrotik-mcp/internal/config"
	"mikrotik-mcp/internal/mcpclient"
	"mikrotik-mcp/internal/orchestrator"
	"mikrotik-mcp/internal/session"
	"mikrotik-mcp/internal/whatsapp"
	"mikrotik-mcp/pkg/logger"
)

func main() {
	// Load .env jika ada
	_ = godotenv.Load()

	cfgPath := "config.yaml"
	if v := os.Getenv("CONFIG_PATH"); v != "" {
		cfgPath = v
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Init logger
	zapLogger, err := logger.New(cfg.Log.Level, cfg.Log.Format)
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer zapLogger.Sync() //nolint:errcheck

	zapLogger.Info("starting mikrotik whatsapp bot",
		zap.String("gowa_url", cfg.WhatsApp.GowaURL),
		zap.String("mcp_server", cfg.Bot.MCPServerURL),
		zap.String("model", cfg.AI.Model),
	)

	// ── SQLite ────────────────────────────────────────────────────────────────
	db, err := sql.Open("sqlite", "./bot.db?_foreign_keys=on")
	if err != nil {
		zapLogger.Fatal("open sqlite", zap.Error(err))
	}
	if err := runMigrations(db); err != nil {
		zapLogger.Fatal("migration failed", zap.Error(err))
	}
	defer db.Close()

	// ── Session Manager ───────────────────────────────────────────────────────
	sessionMgr := session.NewManager(
		session.NewStore(db),
		cfg.Bot.SessionTTL,
		cfg.Bot.MaxHistoryMessages,
		zapLogger,
	)

	// ── MCP Client — connect ke MCP server (SSE mode) ─────────────────────────
	mcpCli, err := mcpclient.NewClient(cfg.Bot.MCPServerURL, zapLogger)
	if err != nil {
		zapLogger.Fatal("connect to MCP server", zap.Error(err))
	}
	defer mcpCli.Close()

	// ── MCP Bridge ────────────────────────────────────────────────────────────
	mcpBridge := bridge.New(mcpCli, zapLogger)
	mcpBridge.SetAuditLogger(bridge.NewAuditLogger(db, zapLogger))
	if err := mcpBridge.RefreshTools(context.Background()); err != nil {
		zapLogger.Fatal("refresh MCP tools", zap.Error(err))
	}

	// ── Z.AI Client ───────────────────────────────────────────────────────────
	zaiClient := zai.NewClient(cfg.AI.APIKey, cfg.AI.BaseURL, cfg.AI.Model, zapLogger)

	// ── Orchestrator ──────────────────────────────────────────────────────────
	orch := orchestrator.New(orchestrator.Config{
		ZAI:          zaiClient,
		Bridge:       mcpBridge,
		Session:      sessionMgr,
		SystemPrompt: cfg.AI.SystemPrompt,
		Model:        cfg.AI.Model,
		MaxTokens:    cfg.AI.MaxTokens,
		Temperature:  cfg.AI.Temperature,
		MaxLoops:     cfg.Bot.MaxFunctionCallLoops,
		ThinkingMode: cfg.AI.ThinkingMode,
	}, zapLogger)

	// ── WhatsApp ──────────────────────────────────────────────────────────────
	sender := whatsapp.NewSender(cfg.WhatsApp.GowaURL, cfg.WhatsApp.GowaDeviceID, cfg.WhatsApp.GowaUsername, cfg.WhatsApp.GowaPassword, zapLogger)
	auth := whatsapp.NewMiddleware(cfg.Bot.AuthorizedUsers)
	handler := whatsapp.NewHandler(orch, sender, auth, cfg.WhatsApp.WebhookSecret, zapLogger)

	// ── HTTP Router ───────────────────────────────────────────────────────────
	r := chi.NewRouter()
	r.Post(cfg.WhatsApp.WebhookPath, handler.HandleWebhook)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"status":"ok","tools":%d}`, mcpBridge.ToolCount())))
	})

	addr := fmt.Sprintf(":%d", cfg.WhatsApp.WebhookPort)
	zapLogger.Info("bot service started",
		zap.String("addr", addr),
		zap.String("webhook_path", cfg.WhatsApp.WebhookPath),
		zap.Int("tools", mcpBridge.ToolCount()),
	)

	// ── Graceful Shutdown ─────────────────────────────────────────────────────
	srv := &http.Server{Addr: addr, Handler: r}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-sigCh:
		zapLogger.Info("shutting down bot service...")
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*1e9)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			zapLogger.Fatal("bot server error", zap.Error(err))
		}
	}

	zapLogger.Info("bot service stopped")
}

func runMigrations(db *sql.DB) error {
	sqlBytes, err := os.ReadFile("migrations/001_sessions.sql")
	if err != nil {
		return fmt.Errorf("read migration file: %w", err)
	}
	if _, err := db.Exec(string(sqlBytes)); err != nil {
		return fmt.Errorf("exec migration: %w", err)
	}
	return nil
}
