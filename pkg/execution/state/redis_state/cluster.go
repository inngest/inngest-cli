package redis_state

import (
	"context"
	"github.com/redis/rueidis"
	"time"
)

type retryClusterDownClient struct {
	r rueidis.Client
}

func (r retryClusterDownClient) B() rueidis.Builder {
	return r.r.B()
}

func (r retryClusterDownClient) do(ctx context.Context, cmd rueidis.Completed, attempts int) rueidis.RedisResult {
	resp := r.r.Do(ctx, cmd)
	if err := resp.Error(); err == nil {
		if ret, ok := rueidis.IsRedisErr(err); ok {
			if ret.IsClusterDown() {
				if attempts == 5 {
					return resp
				}

				time.Sleep(100 * time.Millisecond)
				return r.do(ctx, cmd, attempts+1)
			}
		}
	}

	return resp
}

func (r retryClusterDownClient) Do(ctx context.Context, cmd rueidis.Completed) (resp rueidis.RedisResult) {
	return r.do(ctx, cmd, 0)
}

func (r retryClusterDownClient) DoMulti(ctx context.Context, multi ...rueidis.Completed) (resp []rueidis.RedisResult) {
	return r.r.DoMulti(ctx, multi...)
}

func (r retryClusterDownClient) Receive(ctx context.Context, subscribe rueidis.Completed, fn func(msg rueidis.PubSubMessage)) error {
	return r.r.Receive(ctx, subscribe, fn)
}

func (r retryClusterDownClient) Close() {
	r.r.Close()
}

func (r retryClusterDownClient) DoCache(ctx context.Context, cmd rueidis.Cacheable, ttl time.Duration) (resp rueidis.RedisResult) {
	return r.r.DoCache(ctx, cmd, ttl)

}

func (r retryClusterDownClient) DoMultiCache(ctx context.Context, multi ...rueidis.CacheableTTL) (resp []rueidis.RedisResult) {
	return r.r.DoMultiCache(ctx, multi...)
}

func (r retryClusterDownClient) Dedicated(fn func(rueidis.DedicatedClient) error) (err error) {
	return r.r.Dedicated(fn)
}

func (r retryClusterDownClient) Dedicate() (client rueidis.DedicatedClient, cancel func()) {
	return r.r.Dedicate()
}

func (r retryClusterDownClient) Nodes() map[string]rueidis.Client {
	return r.r.Nodes()
}

func newRetryClusterDownClient(r rueidis.Client) rueidis.Client {
	return &retryClusterDownClient{r: r}
}