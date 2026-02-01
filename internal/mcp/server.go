package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/drewjocham/mongo-migration-tool/internal/config"
	"github.com/drewjocham/mongo-migration-tool/migration"
)

type MCPServer struct {
	mu        sync.RWMutex
	mcpServer *mcp.Server
	engine    *migration.Engine
	db        *mongo.Database
	client    *mongo.Client
	config    *config.Config
	cancel    context.CancelFunc
}

func NewMCPServer() (*MCPServer, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("config load failed: %w", err)
	}

	s := mcp.NewServer(&mcp.Implementation{
		Name:    "mongo-migration",
		Version: "1.0.0",
	}, nil)

	srv := &MCPServer{mcpServer: s, config: cfg}
	srv.registerTools()
	return srv, nil
}

func (s *MCPServer) registerTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "migration_status",
		Description: "Check applied and pending migrations.",
	}, s.handleStatus)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "migration_up",
		Description: "Apply pending migrations.",
	}, s.handleUp)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "migration_down",
		Description: "Roll back migrations.",
	}, s.handleDown)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "migration_create",
		Description: "Generate a new migration file.",
	}, s.handleCreate)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "database_schema",
		Description: "View collections and indexes.",
	}, s.handleSchema)
}

func (s *MCPServer) ensureConnection(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		if err := s.client.Ping(ctx, nil); err == nil {
			return nil
		}
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(s.config.GetConnectionString()))
	if err != nil {
		return err
	}

	s.client = client
	s.db = client.Database(s.config.Database)
	s.engine = migration.NewEngine(s.db, s.config.MigrationsCollection, migration.RegisteredMigrations())
	return nil
}

func (s *MCPServer) Close() error {
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
		if err := client.Disconnect(context.Background()); err != nil {
			errs = append(errs, fmt.Errorf("failed to disconnect mongo client: %w", err))
		}
	}

	return errors.Join(errs...)
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

	return s.Serve(ctx, os.Stdin, os.Stdout)
}

func (s *MCPServer) Serve(ctx context.Context, r io.Reader, w io.Writer) error {
	return s.mcpServer.Run(ctx, &mcp.IOTransport{
		Reader: io.NopCloser(r),
		Writer: nopWriteCloser{Writer: w},
	})
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }
