package keymate

import (
	"context"

	"github.com/numkem/traffikey"
)

type KeymateConnector interface {
	ApplyConfig(ctx context.Context, cfg *traffikey.Config) []error
	ListTargets(ctx context.Context, cfg *traffikey.Config) ([]*traffikey.Target, error)
	ListTargetsByOwner(ctx context.Context, owner string) ([]*traffikey.Target, error)
	DeleteTargetByName(ctx context.Context, target string, prefix string) error

	GetState(ctx context.Context) (*traffikey.Config, error)
	SaveState(ctx context.Context, cfg *traffikey.Config) error
}
