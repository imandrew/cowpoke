package commands

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"cowpoke/internal/domain"
)

// RemoveCommand handles removing Rancher servers from the configuration.
type RemoveCommand struct {
	configRepo domain.ConfigRepository
	logger     *slog.Logger
}

// NewRemoveCommand creates a new remove command.
func NewRemoveCommand(configRepo domain.ConfigRepository, logger *slog.Logger) *RemoveCommand {
	return &RemoveCommand{
		configRepo: configRepo,
		logger:     logger,
	}
}

// RemoveRequest contains the parameters for the remove command.
type RemoveRequest struct {
	ServerURL string
	ServerID  string
}

// Execute runs the remove command.
func (c *RemoveCommand) Execute(ctx context.Context, req RemoveRequest) error {
	switch {
	case req.ServerURL != "":
		// Remove by URL
		c.logger.InfoContext(ctx, "Removing server", "url", req.ServerURL)

		if err := c.configRepo.RemoveServer(ctx, req.ServerURL); err != nil {
			return fmt.Errorf("failed to remove server: %w", err)
		}

		c.logger.InfoContext(ctx, "Successfully removed server", "url", req.ServerURL)
	case req.ServerID != "":
		// Remove by ID
		c.logger.InfoContext(ctx, "Removing server", "id", req.ServerID)

		if err := c.configRepo.RemoveServerByID(ctx, req.ServerID); err != nil {
			return fmt.Errorf("failed to remove server: %w", err)
		}

		c.logger.InfoContext(ctx, "Successfully removed server", "id", req.ServerID)
	default:
		return errors.New("either ServerURL or ServerID must be specified")
	}

	return nil
}
