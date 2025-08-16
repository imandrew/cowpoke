package commands

import (
	"context"
	"fmt"
	"log/slog"

	"cowpoke/internal/domain"
)

// AddCommand handles adding new Rancher servers to the configuration.
type AddCommand struct {
	configRepo domain.ConfigRepository
	logger     *slog.Logger
}

// NewAddCommand creates a new add command.
func NewAddCommand(configRepo domain.ConfigRepository, logger *slog.Logger) *AddCommand {
	return &AddCommand{
		configRepo: configRepo,
		logger:     logger,
	}
}

// AddRequest contains the parameters for the add command.
type AddRequest struct {
	URL      string
	Username string
	AuthType string
}

// Execute runs the add command.
func (c *AddCommand) Execute(ctx context.Context, req AddRequest) error {
	server := domain.ConfigServer{
		URL:      req.URL,
		Username: req.Username,
		AuthType: req.AuthType,
	}

	c.logger.InfoContext(ctx, "Adding new server",
		"id", server.ID(),
		"url", req.URL,
		"username", req.Username,
		"authType", req.AuthType)

	if err := c.configRepo.AddServer(ctx, server); err != nil {
		return fmt.Errorf("failed to add server: %w", err)
	}

	c.logger.InfoContext(ctx, "Successfully added server", "id", server.ID(), "url", req.URL)
	return nil
}
