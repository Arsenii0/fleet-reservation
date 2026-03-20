package ports

import (
	"context"
)

type ResourceMessageSenderPort interface {
	PostMessage(ctx context.Context, topic string, msg interface{}) error
}
