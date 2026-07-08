package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	appconfig "github.com/vk7416/generic-k8s-mcp/internal/config"
	"github.com/vk7416/generic-k8s-mcp/internal/kube"
	"github.com/vk7416/generic-k8s-mcp/internal/mcp"
	"github.com/vk7416/generic-k8s-mcp/internal/tools"
)

const version = "0.1.0"

func main() {
	cfg := appconfig.FromFlags()
	logger := log.New(os.Stderr, "k8s-mcp-server: ", log.LstdFlags)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	clients, err := kube.NewClients(ctx, cfg)
	if err != nil {
		logger.Fatalf("failed to initialize Kubernetes clients: %v", err)
	}

	registry := tools.NewRegistry(ctx, cfg, clients)
	server := &mcp.Server{
		Name:     "generic-k8s-mcp",
		Version:  version,
		Registry: registry,
		Logger:   logger,
	}

	switch cfg.Transport {
	case "stdio", "":
		logger.Printf("starting stdio MCP server mode=%s context=%s namespace=%s readonly=%t", clients.Mode, clients.CurrentContext, clients.DefaultNamespace, cfg.ReadOnly)
		if err := server.RunStdio(os.Stdin, os.Stdout); err != nil {
			logger.Fatalf("stdio server failed: %v", err)
		}
	default:
		_, _ = fmt.Fprintf(os.Stderr, "unsupported transport %q; only stdio is implemented\n", cfg.Transport)
		os.Exit(2)
	}
}
