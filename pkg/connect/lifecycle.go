package connect

import (
	"context"

	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/proto/gen/connect/v1"
)

type ConnectGatewayLifecycleListener interface {
	// OnConnected is called when a new connection is established on the gateway
	OnConnected(ctx context.Context, data *connect.SDKConnectRequestData)

	// OnAuthenticated is called when the established connect has successfully authenticated
	OnAuthenticated(ctx context.Context, auth *state.AuthContext)

	// OnSynced is called when the gateway successfully synced a worker group configuration
	OnSynced(ctx context.Context)

	// OnDisconnected is called when a connection on the gateway is lost
	OnDisconnected(ctx context.Context)
}