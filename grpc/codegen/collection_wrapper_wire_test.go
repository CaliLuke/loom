package codegen

const collectionWrapperMountedHarness = `
func TestMountedCollectionContents(t *testing.T) {
    var calls atomic.Int32
    received := make(chan *rich.Rich, 1)
    endpoints := &rich.Endpoints{Echo: func(_ context.Context, value any) (any, error) {
        calls.Add(1)
        received <- value.(*rich.Rich)
        return value, nil
    }}
    listener := bufconn.Listen(1 << 20)
    host := grpc.NewServer()
    richpb.RegisterRichServer(host, server.New(endpoints, nil))
    done := make(chan error, 1)
    go func() {
        done <- host.Serve(listener)
    }()
    t.Cleanup(func() {
        host.Stop()
        require.NoError(t, <-done)
    })
    conn, err := grpc.NewClient("passthrough:///contents", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
        return listener.Dial()
    }), grpc.WithTransportCredentials(insecure.NewCredentials()))
    require.NoError(t, err)
    t.Cleanup(func() {
        require.NoError(t, conn.Close())
    })
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    remote := richpb.NewRichClient(conn)
    for _, tc := range []struct {
        name string
        value *rich.Rich
        problem string
    }{
        {name: "anonymous minimum", value: &rich.Rich{Req:"r", RequiredItems: rich.Items{}, Nonempty: [][]int{{}}}, problem:"invalid_length"},
        {name: "named minimum", value: &rich.Rich{Req:"r", RequiredItems: rich.Items{}, NamedNonempty: []rich.NonemptyItems{{}}}, problem:"invalid_length"},
        {name: "required wrapper absent", value: &rich.Rich{Req:"r"}, problem:"missing_field"},
        {name: "present empty", value: &rich.Rich{Req:"r", RequiredItems: rich.Items{}, Nested: [][]int{{}}, Mapleaf: map[string][]*rich.Leaf{"k": {}}, Mm: map[string]map[string]int{"k": {}}, Named: []rich.Items{{}}, Maps: []map[string]int{{}}}},
        {name: "populated", value: &rich.Rich{Req:"r", RequiredItems: rich.Items{1}, Nested: [][]int{{1}}, Mapleaf: map[string][]*rich.Leaf{"k": {{Name:"leaf"}}}, Mm: map[string]map[string]int{"k": {"x":1}}, Nonempty: [][]int{{1}}, NamedNonempty: []rich.NonemptyItems{{1}}, Maps: []map[string]int{{"x":1}}}},
    } {
        t.Run(tc.name, func(t *testing.T) {
            before := calls.Load()
            request, err := client.NewProtoEchoRequest(tc.value)
            require.NoError(t, err)
            response, err := remote.Echo(ctx, request)
            if tc.problem != "" {
                require.Equal(t, codes.InvalidArgument, status.Code(err))
                require.Nil(t, response)
                problem, ok := loomgrpc.DecodeError(err).(*loompb.ErrorResponse)
                require.True(t, ok)
                require.Equal(t, tc.problem, problem.Name)
                if tc.problem == "invalid_length" {
                    require.Equal(t, "validation error", status.Convert(err).Message())
                } else {
                    require.Contains(t, status.Convert(err).Message(), "required_items")
                }
                require.Equal(t, before, calls.Load())
                return
            }
            require.NoError(t, err)
            require.Equal(t, before+1, calls.Load())
            select {
            case actual := <-received:
                require.Equal(t, tc.value, actual)
            case <-ctx.Done():
                require.NoError(t, ctx.Err())
            }
            require.NoError(t, client.ValidateEchoResponse(response))
            result, err := client.NewEchoResult(response)
            require.NoError(t, err)
            require.Equal(t, tc.value, result)
        })
    }
}
`
