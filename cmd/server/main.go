package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"

	"mikrotik-mcp/internal/config"
	mkclient "mikrotik-mcp/internal/mikrotik"
	"mikrotik-mcp/internal/usecase"
	"mikrotik-mcp/pkg/logger"
	"mikrotik-mcp/tools"
)

func main() {
	// Load config
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

	zapLogger.Info("starting mikrotik-mcp",
		zap.String("transport", cfg.MCP.Transport),
		zap.String("router", cfg.MikroTik.Host),
		zap.Bool("read_only", cfg.MCP.ReadOnly),
	)

	// Init MikroTik client
	mtClient := mkclient.NewClient(mkclient.Config{
		Host:              cfg.MikroTik.Host,
		Port:              cfg.MikroTik.Port,
		Username:          cfg.MikroTik.Username,
		Password:          cfg.MikroTik.Password,
		UseTLS:            cfg.MikroTik.UseTLS,
		ReconnectInterval: cfg.MikroTik.ReconnectInterval,
		Timeout:           cfg.MikroTik.Timeout,
	}, zapLogger)

	// Connect to router
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := mtClient.Connect(ctx); err != nil {
		zapLogger.Fatal("failed to connect to mikrotik", zap.Error(err))
	}
	defer mtClient.Close()

	// Wire repositories
	ipPoolRepo := mkclient.NewIPPoolRepository(mtClient)
	firewallRepo := mkclient.NewFirewallRepository(mtClient)
	interfaceRepo := mkclient.NewInterfaceRepository(mtClient)
	hotspotRepo := mkclient.NewHotspotRepository(mtClient)
	queueRepo := mkclient.NewQueueRepository(mtClient)
	systemRepo := mkclient.NewSystemRepository(mtClient)

	// Wire use cases
	ipPoolUC := usecase.NewIPPoolUseCase(ipPoolRepo, zapLogger)
	firewallUC := usecase.NewFirewallUseCase(firewallRepo, zapLogger)
	interfaceUC := usecase.NewInterfaceUseCase(interfaceRepo, zapLogger)
	hotspotUC := usecase.NewHotspotUseCase(hotspotRepo, zapLogger)
	queueUC := usecase.NewQueueUseCase(queueRepo, zapLogger)
	systemUC := usecase.NewSystemUseCase(systemRepo, zapLogger)

	// Create MCP server
	mcpServer := server.NewMCPServer(
		"mikrotik-mcp",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Register all tools
	tools.RegisterAll(mcpServer, tools.Dependencies{
		IPPool:    ipPoolUC,
		Firewall:  firewallUC,
		Interface: interfaceUC,
		Hotspot:   hotspotUC,
		Queue:     queueUC,
		System:    systemUC,
		ReadOnly:  cfg.MCP.ReadOnly,
	})

	zapLogger.Info("all tools registered")

	// Start transport
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	switch cfg.MCP.Transport {
	case "sse":
		addr := fmt.Sprintf(":%d", cfg.MCP.Port)
		zapLogger.Info("starting SSE transport", zap.String("addr", addr))

		sseServer := server.NewSSEServer(mcpServer)
		errCh := make(chan error, 1)
		go func() {
			errCh <- sseServer.Start(addr)
		}()

		select {
		case <-sigCh:
			zapLogger.Info("shutting down...")
			_ = sseServer.Shutdown(ctx)
		case err := <-errCh:
			if err != nil {
				zapLogger.Fatal("SSE server error", zap.Error(err))
			}
		}

	default: // stdio
		zapLogger.Info("starting stdio transport")
		stdioServer := server.NewStdioServer(mcpServer)

		errCh := make(chan error, 1)
		go func() {
			errCh <- stdioServer.Listen(ctx, os.Stdin, os.Stdout)
		}()

		select {
		case <-sigCh:
			zapLogger.Info("shutting down...")
			cancel()
		case err := <-errCh:
			if err != nil {
				zapLogger.Fatal("stdio server error", zap.Error(err))
			}
		}
	}

	zapLogger.Info("mikrotik-mcp stopped")
}
