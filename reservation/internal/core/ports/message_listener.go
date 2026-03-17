package ports

import (
	"context"
)

// MessageListener defines the interface for a message listener adaptor.
type MessageListener interface {
	// Run starts the message consuming process. It should be non-blocking.
	Run(ctx context.Context)

	// Close shuts down the message listener and cleans up resources.
	Close() error
}
