package connectdriver

import (
	"context"
	"errors"
	"github.com/inngest/inngest/pkg/connect/pubsub"
	"github.com/inngest/inngest/pkg/execution/driver/httpdriver"
	connect_sdk "github.com/inngest/inngestgo/connect"
	"net/http"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/syscode"
)

type executor struct {
	forwarder pubsub.RequestForwarder
}

// RuntimeType fulfills the inngest.Runtime interface.
func (e executor) RuntimeType() string {
	return "connect"
}

func (e executor) Execute(ctx context.Context, sl sv2.StateLoader, s sv2.Metadata, item queue.Item, edge inngest.Edge, step inngest.Step, idx, attempt int) (*state.DriverResponse, error) {

	input, err := driver.MarshalV1(ctx, sl, s, step, idx, "", attempt)
	if err != nil {
		return nil, err
	}

	return ProxyRequest(ctx, e.forwarder, httpdriver.Request{
		Input: input,
		Edge:  edge,
		Step:  step,
	})
}

// ProxyRequest proxies the request to the SDK over a long-lived connection with the given input.
func ProxyRequest(ctx context.Context, forwarder pubsub.RequestForwarder, r httpdriver.Request) (*state.DriverResponse, error) {
	requestToForward := connect_sdk.GatewayMessageTypeExecutorRequestData{
		// TODO Find out if we can supply this in a better way. We still use the URL concept a lot,
		// even though this has no meaning in connect.
		FunctionSlug: r.URL.Query().Get("fnId"),
		RequestBytes: r.Input,
	}
	// If we have a generator step name, ensure we add the step ID parameter
	if r.Edge.IncomingGeneratorStep != "" {
		requestToForward.StepId = &r.Edge.IncomingGeneratorStep
	} else {
		requestToForward.StepId = &r.Edge.Incoming
	}

	resp, err := do(ctx, forwarder, requestToForward)
	if err != nil {
		return nil, err
	}

	return httpdriver.HandleHttpResponse(ctx, r, resp)
}

func do(ctx context.Context, forwarder pubsub.RequestForwarder, data connect_sdk.GatewayMessageTypeExecutorRequestData) (*httpdriver.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, consts.MaxFunctionTimeout)
	defer cancel()

	// TODO Include trace headers
	// Add `traceparent` and `tracestate` headers to the request from `ctx`
	// itrace.UserTracer().Propagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	pre := time.Now()
	resp, err := forwarder.Proxy(ctx, data)
	dur := time.Since(pre)

	// TODO Check if we need some of the request error handling logic from httpdriver.do()

	// Return gateway-handled errors like  syscode.CodeOutputTooLarge
	var sysErr *syscode.Error
	{
		syscodeError := &syscode.Error{}
		if errors.As(err, &syscodeError) {
			sysErr = syscodeError
		}
	}

	// TODO Should be handled above, verify this
	//// Read 1 extra byte above the max so that we can check if the response is
	//// too large
	//byt, err := io.ReadAll(io.LimitReader(resp.Body, consts.MaxBodySize+1))
	//if err != nil {
	//	return nil, fmt.Errorf("error reading response body: %w", err)
	//}
	//var sysErr *syscode.Error
	//if len(byt) > consts.MaxBodySize {
	//	sysErr = &syscode.Error{Code: syscode.CodeOutputTooLarge}
	//
	//	// Override the output so the user sees the syserrV in the UI rather
	//	// than a JSON parsing error
	//	byt, _ = json.Marshal(sysErr.Code)
	//}

	// Check the retry status from the headers and versions.
	noRetry := !httpdriver.ShouldRetry(int(resp.Status), resp.NoRetry, resp.SdkVersion)

	// Extract the retry at header if it hasn't been set explicitly in streaming.
	var retryAtStr *string
	if after := resp.RetryAfter; retryAtStr == nil && after != "" {
		retryAtStr = &after
	}
	var retryAt *time.Time
	if retryAtStr != nil {
		if at, err := httpdriver.ParseRetry(*retryAtStr); err == nil {
			retryAt = &at
		}
	}

	// TODO connect is only implemented by SDKs, but we can include a flag in the proxied resp as well...
	isSDK := true

	return &httpdriver.Response{
		Body:           resp.Body,
		StatusCode:     int(resp.Status),
		Duration:       dur,
		RetryAt:        retryAt,
		NoRetry:        noRetry,
		RequestVersion: 0, // not supported by go sdk even for http
		IsSDK:          isSDK,
		Sdk:            resp.SdkVersion,
		Header:         http.Header{}, // not supported by connect
		SysErr:         sysErr,
	}, err

}