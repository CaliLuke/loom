package admission_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/admission"
	loom "github.com/CaliLuke/loom/pkg"
)

func TestConcurrencyConfiguration(t *testing.T) {
	for _, capacity := range []int{-1, 0} {
		c, err := admission.NewConcurrency(capacity)
		require.Error(t, err)
		require.Nil(t, c)
	}
	c, err := admission.NewConcurrency(2)
	require.NoError(t, err)
	for _, capacity := range []int{-1, 0} {
		require.Error(t, c.SetCapacity(capacity))
		require.Equal(t, 2, c.Snapshot().Capacity)
	}
	require.Panics(t, func() {
		c.Middleware(nil)
	})
	var zero admission.Concurrency
	require.Panics(t, func() {
		zero.Middleware(func(context.Context, any) (any, error) {
			return nil, nil
		})
	})
}

func TestConcurrencyReleasesOnEveryExit(t *testing.T) {
	serviceError := errors.New("service failure")
	for _, outcome := range []string{"success", "error", "panic"} {
		t.Run(outcome, func(t *testing.T) {
			c, err := admission.NewConcurrency(1)
			require.NoError(t, err)
			type traceKey struct{}
			ctx, cancel := context.WithDeadline(context.WithValue(context.Background(), traceKey{}, "trace"), time.Now().Add(time.Minute))
			defer cancel()
			request := new(int)
			ep := c.Middleware(func(got context.Context, req any) (any, error) {
				require.Same(t, ctx, got)
				require.Same(t, request, req)
				require.Equal(t, 1, c.Snapshot().InFlight)
				switch outcome {
				case "error":
					return request, serviceError
				case "panic":
					panic("service panic")
				default:
					return request, nil
				}
			})
			for range 2 {
				if outcome == "panic" {
					require.PanicsWithValue(t, "service panic", func() {
						_, err := ep(ctx, request)
						require.NoError(t, err)
					})
				} else {
					result, err := ep(ctx, request)
					require.Same(t, request, result)
					if outcome == "error" {
						require.ErrorIs(t, err, serviceError)
					} else {
						require.NoError(t, err)
					}
				}
				require.Zero(t, c.Snapshot().InFlight)
			}
			require.Equal(t, uint64(2), c.Snapshot().Admitted)
		})
	}
}

func TestConcurrencyCancellationRetainsPermit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c, err := admission.NewConcurrency(1)
		require.NoError(t, err)
		release := make(chan struct{})
		unblock := sync.OnceFunc(func() {
			close(release)
		})
		defer unblock()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ep := c.Middleware(func(ctx context.Context, _ any) (any, error) {
			<-ctx.Done()
			<-release
			return nil, ctx.Err()
		})
		go func() {
			_, err := ep(ctx, nil)
			assert.ErrorIs(t, err, context.Canceled)
		}()
		synctest.Wait()
		cancel()
		synctest.Wait()
		require.Equal(t, 1, c.Snapshot().InFlight)
		_, err = ep(context.Background(), nil)
		requireOverloaded(t, err)
		_, err = ep(ctx, nil)
		require.ErrorIs(t, err, context.Canceled)
		expired, stop := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer stop()
		_, err = ep(expired, nil)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Equal(t, admission.ConcurrencySnapshot{Capacity: 1, InFlight: 1, Admitted: 1, Rejected: 1, Canceled: 2}, c.Snapshot())
		unblock()
		synctest.Wait()
		require.Zero(t, c.Snapshot().InFlight)
		// Once the canceled invocation exits, its permit can be reused.
		_, err = c.Middleware(func(context.Context, any) (any, error) {
			return "recovered", nil
		})(context.Background(), nil)
		require.NoError(t, err)
		require.Equal(t, uint64(2), c.Snapshot().Admitted)
	})
}

func TestConcurrencySimultaneousCalls(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c, err := admission.NewConcurrency(4)
		require.NoError(t, err)
		start, release := make(chan struct{}), make(chan struct{})
		defer close(release)
		ep := c.Middleware(func(context.Context, any) (any, error) {
			<-release
			return nil, nil
		})
		for range 20 {
			go func() {
				<-start
				_, err := ep(context.Background(), nil)
				if err != nil {
					assertOverloaded(t, err)
				}
			}()
		}
		close(start)
		synctest.Wait()
		require.Equal(t, admission.ConcurrencySnapshot{Capacity: 4, InFlight: 4, Admitted: 4, Rejected: 16}, c.Snapshot())
	})
}

func TestConcurrencyResizeRetainsActiveCalls(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c, err := admission.NewConcurrency(2)
		require.NoError(t, err)
		finish := []chan struct{}{make(chan struct{}), make(chan struct{}), make(chan struct{})}
		ep := c.Middleware(func(_ context.Context, request any) (any, error) {
			<-finish[request.(int)]
			return nil, nil
		})
		start := func(i int) {
			go func() {
				_, err := ep(context.Background(), i)
				assert.NoError(t, err)
			}()
			synctest.Wait()
		}
		start(0)
		start(1)
		require.NoError(t, c.SetCapacity(3))
		start(2)
		require.NoError(t, c.SetCapacity(1))
		require.NoError(t, c.SetCapacity(1))
		require.Error(t, c.SetCapacity(0))
		for _, done := range finish {
			_, err := ep(context.Background(), 0)
			requireOverloaded(t, err)
			close(done)
			synctest.Wait()
		}
		_, err = ep(context.Background(), 0)
		require.NoError(t, err)
		require.Equal(t, admission.ConcurrencySnapshot{Capacity: 1, Admitted: 4, Rejected: 3}, c.Snapshot())
	})
}

func TestConcurrencyUpdateCompletionAndCancellationRace(t *testing.T) {
	c, err := admission.NewConcurrency(4)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	start := make(chan struct{})
	entered, finish := make(chan struct{}), make(chan struct{})
	unblock := sync.OnceFunc(func() {
		close(finish)
	})
	defer unblock()
	var workers sync.WaitGroup
	ep := c.Middleware(func(_ context.Context, request any) (any, error) {
		if request == "held" {
			close(entered)
			<-finish
		}
		return nil, nil
	})
	workers.Go(func() {
		_, err := ep(ctx, "held")
		assert.NoError(t, err)
	})
	<-entered
	require.Equal(t, 1, c.Snapshot().InFlight)
	// A permit is already held before updates, cancellation and its release
	// become runnable. Cancellation cannot avoid the completion/update race.
	for i := range 100 {
		workers.Go(func() {
			<-start
			assert.NoError(t, c.SetCapacity(i%4+1))
			_, err := ep(ctx, nil)
			if err != nil && !errors.Is(err, context.Canceled) {
				assertOverloaded(t, err)
			}
			assert.Error(t, c.SetCapacity(-1))
		})
	}
	workers.Go(func() {
		<-start
		cancel()
	})
	workers.Go(func() {
		<-start
		unblock()
	})
	close(start)
	workers.Wait()
	snapshot := c.Snapshot()
	require.Zero(t, snapshot.InFlight)
	require.GreaterOrEqual(t, snapshot.Admitted, uint64(1))
	require.Equal(t, uint64(101), snapshot.Admitted+snapshot.Rejected+snapshot.Canceled)
	require.GreaterOrEqual(t, snapshot.Capacity, 1)
	require.LessOrEqual(t, snapshot.Capacity, 4)
}

func requireOverloaded(t *testing.T, err error) {
	t.Helper()
	var serviceError *loom.ServiceError
	require.ErrorAs(t, err, &serviceError)
	require.Equal(t, "overloaded", serviceError.Name)
	require.True(t, serviceError.Temporary)
}

func assertOverloaded(t *testing.T, err error) {
	t.Helper()
	var serviceError *loom.ServiceError
	if assert.ErrorAs(t, err, &serviceError) {
		assert.Equal(t, "overloaded", serviceError.Name)
		assert.True(t, serviceError.Temporary)
	}
}
