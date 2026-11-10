package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTypeLayoutsSnapshotAndOverrides(t *testing.T) {
	context := &AttributeContext{
		JSONPresenceTypes:       map[string]bool{"named": true, "json_only": false},
		PresencePointerTypes:    map[string]bool{"named": true, "pointer_only": false},
		PresenceUseDefaultTypes: map[string]bool{"named": false, "default_only": true},
	}
	context.TypeLayouts = CaptureTypeLayouts(context)
	for _, test := range []struct {
		key  string
		want TypeLayout
	}{
		{"named", TypeLayout{JSONPresence: true, Pointer: true, HasJSONPresence: true, HasPointer: true, HasUseDefault: true}},
		{"json_only", TypeLayout{HasJSONPresence: true}},
		{"pointer_only", TypeLayout{HasPointer: true}},
		{"default_only", TypeLayout{UseDefault: true, HasUseDefault: true}},
		{"missing", TypeLayout{}},
	} {
		t.Run(test.key, func(t *testing.T) {
			require.Equal(t, test.want, context.TypeLayouts.Lookup(test.key))
		})
	}
	context.JSONPresenceTypes["named"] = false
	context.PresencePointerTypes["named"] = false
	context.PresenceUseDefaultTypes["named"] = true
	context.JSONPresenceTypes["added"] = false
	want := TypeLayout{UseDefault: true, HasJSONPresence: true, HasPointer: true, HasUseDefault: true}
	require.Equal(t, want, context.TypeLayout("named"))
	require.True(t, context.TypeLayouts.Lookup("named").JSONPresence, "snapshot is detached")
	require.False(t, context.TypeLayouts.Lookup("added").HasJSONPresence)
	require.Equal(t, TypeLayout{HasJSONPresence: true}, context.TypeLayout("added"))
	require.Same(t, context.TypeLayouts, context.Dup().TypeLayouts)
	recaptured := CaptureTypeLayouts(context)
	require.Equal(t, want, recaptured.Lookup("named"))
	presence, pointers, defaults := recaptured.maps()
	presence["named"], pointers["named"], defaults["named"] = true, true, false
	require.Equal(t, want, recaptured.Lookup("named"), "recapture maps are detached")
	require.Equal(t, recaptured.Identity(), CaptureTypeLayouts(context).Identity())
	require.NotEqual(t, context.TypeLayouts.Identity(), recaptured.Identity())
}
