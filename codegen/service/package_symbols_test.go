package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testdata"
)

func TestServicePackageSymbolsDoNotCollide(t *testing.T) {
	root := codegen.RunDSL(t, testdata.ServiceReservedSymbolsDSL)
	data := NewServicesData(root).Get("symbols")
	reserved := map[string]bool{
		"Client": true, "Endpoints": true, "Service": true,
		"APIName": true, "APIVersion": true, "ServiceName": true,
		"MethodNames": true, "NewClient": true, "NewEndpoints": true,
	}
	for _, typ := range data.userTypes {
		t.Run(typ.Type.Name(), func(t *testing.T) {
			require.False(t, reserved[typ.VarName], "authored type %q collides with fixed generated declaration %q", typ.Type.Name(), typ.VarName)
		})
	}
}

func TestConditionalServicePackageSymbols(t *testing.T) {
	for _, tc := range []struct {
		mode     string
		reserved []string
	}{
		{mode: "plain"},
		{mode: "secure", reserved: []string{"Authorizer"}},
		{mode: "sse", reserved: []string{"Stream", "Event"}},
		{mode: "websocket", reserved: []string{"Stream"}},
		{mode: "interceptors", reserved: []string{"ServerInterceptors", "ClientInterceptors"}},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				testdata.ServiceConditionalSymbolsDSL(tc.mode)
			})
			data := NewServicesData(root).Get("symbols")
			for _, typ := range data.userTypes {
				reserved := false
				for _, name := range tc.reserved {
					reserved = reserved || name == typ.Type.Name()
				}
				t.Run(typ.Type.Name(), func(t *testing.T) {
					if reserved {
						require.NotEqual(t, typ.Type.Name(), typ.VarName, "fixed %s declaration needs ownership", typ.Type.Name())
					} else {
						require.Equal(t, typ.Type.Name(), typ.VarName, "an absent capability must not rename %s", typ.Type.Name())
					}
				})
			}
		})
	}
}

func TestFixedPackageSymbolsDoNotRenameMethods(t *testing.T) {
	root := codegen.RunDSL(t, testdata.ServiceFixedSpellingMethodsDSL)
	data := NewServicesData(root).Get("symbols")
	for _, method := range data.Methods {
		require.Equal(t, method.Name, method.VarName)
	}
}

func TestMethodAndPackageTypeNamespacesAreIndependent(t *testing.T) {
	root := codegen.RunDSL(t, testdata.ServiceMethodTypeCouplingDSL)
	data := NewServicesData(root).Get("symbols")
	require.Len(t, data.Methods, 2)
	method := data.Methods[1]
	t.Logf("method=%s server=%s client=%s input=%s field=%s serverAdapter=%s clientAdapter=%s", method.VarName, method.ServerStream.Interface, method.ClientStream.Interface, method.ServerStream.EndpointStruct, method.EndpointField, method.ServerStream.VarName, method.ClientStream.VarName)
	require.Equal(t, "Foo", method.VarName, "package types do not occupy the receiver/interface method namespace")
	for _, typ := range data.userTypes {
		require.NotEqual(t, method.ServerStream.Interface, typ.VarName)
		require.NotEqual(t, method.ClientStream.Interface, typ.VarName)
		require.NotEqual(t, method.ServerStream.EndpointStruct, typ.VarName)
	}
}

func TestServiceMemberNamespaceControls(t *testing.T) {
	root := codegen.RunDSL(t, testdata.ServiceMemberFieldsDSL)
	data := NewServicesData(root).Get("symbols")
	for name, want := range map[string]string{"foo": "Foo", "fooEndpoint": "FooEndpoint", "Use": "UseEndpoint", "HandleStream": "HandleStream"} {
		require.Equal(t, want, data.Method(name).VarName)
	}
	require.Equal(t, "FooEndpoint2", data.Method("foo").EndpointField, "client fields must avoid the client's real methods")
}

func TestServicePackageHelperReferences(t *testing.T) {
	for _, tc := range []struct {
		name   string
		design func()
		want   map[string]string
	}{
		{"fixed", testdata.ServiceReservedSymbolsDSL, map[string]string{"Client": "Client2", "Client2": "Client22"}},
		{"carriers", testdata.ServiceCarrierSymbolsDSL, map[string]string{"UploadRequestData": "UploadRequestData2", "DownloadResponseData": "DownloadResponseData2", "ServeFileResponseData": "ServeFileResponseData2", "ServeResponseData": "ServeResponseData2", "WrapFooEndpoint": "WrapFooEndpoint2", "WrapFooClientEndpoint": "WrapFooClientEndpoint3", "Foo": "Foo"}},
		{"sse-event", testdata.ServiceSSEEventSymbolsDSL, map[string]string{"WatchEvent": "WatchEvent2"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := codegen.RunDSL(t, tc.design)
			data := NewServicesData(root).Get("symbols")
			for _, typ := range data.userTypes {
				if want, ok := tc.want[typ.Name]; ok {
					require.Equal(t, want, typ.VarName)
					require.Equal(t, "*"+want, typ.Ref)
				}
			}
			if tc.name == "carriers" {
				require.Equal(t, "UploadRequestData", data.Method("upload").RequestStruct)
				require.Equal(t, "DownloadResponseData", data.Method("download").ResponseStruct)
				require.Equal(t, "ServeResponseData", data.Method("serve").ResponseStruct)
				require.Equal(t, "ServeFileResponseData", data.Method("serve").FileResponseStruct)
				require.Equal(t, "WrapFooClientEndpoint", data.Method("foo").ClientInterceptorWrapper(false))
				require.Equal(t, "WrapFooClientEndpoint2", data.Method("fooClient").ServerInterceptorWrapper)
			}
			if tc.name == "sse-event" {
				require.Equal(t, "WatchEvent", data.Method("watch").EventName)
				require.Equal(t, "*WatchEvent2", data.Method("watch").ServerStream.SendTypeRef)
			}
		})
	}
}
