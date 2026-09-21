package codegen

// The independent adversarial design exercises changes between admission and
// service invocation, rather than only checking that a rejecting evaluator runs.
const authorizationAdversarialHarness = `
func TestAuthorizationMutationBoundaries(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(context.Context, any, *accessEvaluator) (context.Context, any)
	}{
		{"target", func(ctx context.Context, req any, _ *accessEvaluator) (context.Context, any) {
			req.(*documents.UpdatePayload).DocumentID = "denied"
			return ctx, req
		}},
		{"credential", func(ctx context.Context, req any, _ *accessEvaluator) (context.Context, any) {
			req.(*documents.UpdatePayload).Token = "invalid"
			return ctx, req
		}},
		{"payload type", func(ctx context.Context, _ any, _ *accessEvaluator) (context.Context, any) {
			return ctx, "wrong type"
		}},
		{"revocation", func(ctx context.Context, req any, a *accessEvaluator) (context.Context, any) {
			a.denied = true
			return ctx, req
		}},
		{"cancellation", func(ctx context.Context, req any, _ *accessEvaluator) (context.Context, any) {
			canceled, cancel := context.WithCancel(ctx)
			cancel()
			return canceled, req
		}},
	}
	for _, tc := range cases {
		for _, interceptor := range []bool{false, true} {
			t.Run(tc.name+map[bool]string{false: "/middleware", true: "/interceptor"}[interceptor], func(t *testing.T) {
				s := &documentService{}
				a := &accessEvaluator{}
				mutate := func(ctx context.Context, req any, next loom.Endpoint) (any, error) {
					ctx, req = tc.mutate(ctx, req, a)
					return next(ctx, req)
				}
				i := &documentInterceptors{}
				if interceptor {
					i.apply = mutate
				}
				e := documents.NewEndpoints(s, a, i)
				if !interceptor {
					e.Use(func(next loom.Endpoint) loom.Endpoint {
						return func(ctx context.Context, req any) (any, error) {
							return mutate(ctx, req, next)
						}
					})
				}
				_, err := e.Update(context.Background(), &documents.UpdatePayload{Token: "valid", DocumentID: "allowed"})
				require.Error(t, err)
				require.Zero(t, s.calls)
			})
		}
	}
}

func TestAuthorizationConjunctionAndInterceptorCache(t *testing.T) {
	for _, tc := range []struct {
		first, second bool
		audits        int
	}{{true, false, 0}, {false, true, 1}} {
		s := &documentService{}
		a := &accessEvaluator{denied: tc.first, auditDenied: tc.second}
		i := &documentInterceptors{apply: func(context.Context, any, loom.Endpoint) (any, error) {
			t.Error("interceptor ran before authorization succeeded")
			return "cached", nil
		}}
		e := documents.NewEndpoints(s, a, i)
		_, err := e.Update(context.Background(), &documents.UpdatePayload{Token: "valid", DocumentID: "allowed"})
		require.Error(t, err)
		require.Equal(t, tc.audits, a.audits)
		require.Zero(t, s.calls)
	}
}

func TestAuthorizationAuthContextFailure(t *testing.T) {
	for _, replace := range []bool{false, true} {
		s := &documentService{}
		a := &accessEvaluator{}
		ctx, cancel := context.WithCancel(context.Background())
		auth := func(ctx context.Context, _ string, _ *security.JWTScheme) (context.Context, error) {
			cancel()
			if replace {
				return context.Background(), nil
			}
			return ctx, nil
		}
		ep := documents.NewUpdateEndpoint(s, a, auth)
		_, err := ep(ctx, &documents.UpdatePayload{Token: "valid", DocumentID: "allowed"})
		require.ErrorIs(t, err, context.Canceled)
		require.Zero(t, a.checks)
		require.Zero(t, s.calls)
	}
	s := &documentService{}
	a := &accessEvaluator{}
	ep := documents.NewUpdateEndpoint(s, a, func(context.Context, string, *security.JWTScheme) (context.Context, error) {
		return nil, nil
	})
	_, err := ep(context.Background(), &documents.UpdatePayload{Token: "valid", DocumentID: "allowed"})
	require.Error(t, err)
	require.Zero(t, a.checks)
	require.Zero(t, s.calls)
}

func TestAuthorizationMiddlewareCannotDiscardCancellation(t *testing.T) {
	for _, detached := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		s := &documentService{}
		e := documents.NewEndpoints(s, &accessEvaluator{}, &documentInterceptors{})
		e.Use(func(next loom.Endpoint) loom.Endpoint {
			return func(ctx context.Context, req any) (any, error) {
				cancel()
				if detached {
					return next(context.WithoutCancel(ctx), req)
				}
				return next(context.Background(), req)
			}
		})
		_, err := e.Update(ctx, &documents.UpdatePayload{Token: "valid", DocumentID: "allowed"})
		require.Error(t, err)
		require.Zero(t, s.calls)
	}
}
`
