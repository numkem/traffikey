package keymate

import (
	"context"

	traefikkeymate "github.com/numkem/traefik-keymate"
)

type KeymateConnector interface {
	ApplyConfig(ctx context.Context, cfg *traefikkeymate.Config) []error
}
