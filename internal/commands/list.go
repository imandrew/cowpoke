package commands

import (
	"context"
	"fmt"
	"log/slog"

	"cowpoke/internal/domain"
)

// ListCommand handles listing configured Rancher servers.
type ListCommand struct {
	configRepo domain.ConfigRepository
	logger     *slog.Logger
}

// NewListCommand creates a new list command.
func NewListCommand(configRepo domain.ConfigRepository, logger *slog.Logger) *ListCommand {
	return &ListCommand{
		configRepo: configRepo,
		logger:     logger,
	}
}

// ListRequest contains the parameters for the list command.
type ListRequest struct {
	OutputFormat string // "table", "json", "yaml"
	Verbose      bool
}

// ListResult contains the result of the list command.
type ListResult struct {
	Servers []domain.ConfigServer
	Count   int
}

// Execute runs the list command.
func (c *ListCommand) Execute(ctx context.Context, _ ListRequest) (*ListResult, error) {
	c.logger.DebugContext(ctx, "Listing configured servers")

	servers, err := c.configRepo.GetServers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get servers: %w", err)
	}

	result := &ListResult{
		Servers: servers,
		Count:   len(servers),
	}

	c.logger.InfoContext(ctx, "Retrieved server list", "count", len(servers))
	return result, nil
}
