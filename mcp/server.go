package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/drewjocham/mongo-migration-tool/internal/config"
	"github.com/drewjocham/mongo-migration-tool/internal/migration"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MCPServer struct {
	mu        sync.RWMutex
	mcpServer *mcp.Server
	engine    *migration.Engine
	db        *mongo.Database
	client    *mongo.Client
	config    *config.Config
	cancel    context.CancelFunc
	logger    *slog.Logger
}

func NewMCPServer(cfg *config.Config, logger *slog.Logger) (*MCPServer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	s := mcp.NewServer(&mcp.Implementation{
		Name:    "mongo-migration",
		Version: "1.0.0",
	}, nil)

	srv := &MCPServer{
		mcpServer: s,
		config:    cfg,
		logger:    logger,
	}

	srv.registerTools()
	return srv, nil
}

func (s *MCPServer) ensureConnection(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		if err := s.client.Ping(ctx, nil); err == nil {
			return nil
		}
	}

	client, err := mongo.Connect(options.Client().ApplyURI(s.config.MongoURL))
	if err != nil {
		return fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	s.client = client
	s.db = client.Database(s.config.Database)
	s.engine = migration.NewEngine(s.db, s.config.MigrationsCollection, migration.RegisteredMigrations())

	s.logger.Info("connected to mongodb", "database", s.config.Database)
	return nil
}

func (s *MCPServer) Start() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		stop()
		return fmt.Errorf("mcp server already running")
	}
	s.cancel = stop
	s.mu.Unlock()

	defer func() {
		stop()
		s.mu.Lock()
		s.cancel = nil
		s.mu.Unlock()
	}()

	s.logger.Info("starting mcp server")
	return s.Serve(ctx, os.Stdin, os.Stdout)
}

func (s *MCPServer) Serve(ctx context.Context, r io.Reader, w io.Writer) error {
	return s.mcpServer.Run(ctx, &mcp.IOTransport{
		Reader: io.NopCloser(r),
		Writer: nopWriteCloser{Writer: w},
	})
}

func (s *MCPServer) Close(ctx context.Context) error {
	s.mu.Lock()
	client := s.client
	s.client = nil
	cancel := s.cancel
	s.cancel = nil
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	var errs []error
	if client != nil {
		if err := client.Disconnect(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to disconnect mongo client: %w", err))
		}
	}

	return errors.Join(errs...)
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }
