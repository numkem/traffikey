package keymate

import (
	"context"

	"github.com/numkem/traffikey"
)

type KeymateConnector interface {
	ApplyConfig(ctx context.Context, cfg *traffikey.Config) []error
}
