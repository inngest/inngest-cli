package state

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	connpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/redis/rueidis"
)

type redisConnectionStateManager struct {
	client rueidis.Client
}

func NewRedisConnectionStateManager(client rueidis.Client) *redisConnectionStateManager {
	return &redisConnectionStateManager{
		client: client,
	}
}

func (r redisConnectionStateManager) SetRequestIdempotency(ctx context.Context, appId uuid.UUID, requestId string) error {
	idempotencyKey := fmt.Sprintf("{%s}:idempotency:%s", appId, requestId)
	res := r.client.Do(
		ctx,
		r.client.B().Set().Key(idempotencyKey).Value("1").Nx().Ex(time.Second*10).Build(),
	)
	set, err := res.AsBool()
	if (err == nil || rueidis.IsRedisNil(err)) && !set {
		return ErrIdempotencyKeyExists
	}
	if err != nil {
		return fmt.Errorf("could not set idempotency key: %w", err)
	}

	return nil
}

func (r *redisConnectionStateManager) GetConnectionsByEnvID(ctx context.Context, wsID uuid.UUID) ([]*connpb.ConnMetadata, error) {
	return nil, notImplementedError
}

func (r *redisConnectionStateManager) GetConnectionsByAppID(ctx context.Context, appID uuid.UUID) ([]*connpb.ConnMetadata, error) {
	return nil, notImplementedError
}

func (r *redisConnectionStateManager) AddConnection(ctx context.Context, wsID uuid.UUID, meta *connpb.ConnMetadata) error {
	return notImplementedError
}

func (r *redisConnectionStateManager) DeleteConnection(ctx context.Context, connID string) error {
	return notImplementedError
}

//
// Lifecycle hooks
//

func (r *redisConnectionStateManager) OnConnected(ctx context.Context, data *connpb.SDKConnectRequestData) {
}

func (r *redisConnectionStateManager) OnAuthenticated(ctx context.Context, auth *AuthContext) {
}

func (r *redisConnectionStateManager) OnSynced(ctx context.Context) {
}

func (r *redisConnectionStateManager) OnDisconnected(ctx context.Context) {
}