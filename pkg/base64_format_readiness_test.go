package loom

import (
	"encoding/json/v2"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBase64FormatAdmission checks the shared format seam before an imported
// string can preserve encoded constraints without accepting arbitrary strings.
func TestBase64FormatAdmission(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
		valid bool
	}{
		{"empty", "", true},
		{"one byte", "Zg==", true},
		{"two bytes", "aGk=", true},
		{"complete quantum", "Zm9v", true},
		{"multiple quanta", "Zm9vYmFy", true},
		{"pad bit alias", "aGl=", true},
		{"missing padding", "aGk", false},
		{"extra padding", "aGk==", false},
		{"internal padding", "a=k=", false},
		{"newline", "aGk=\n", false},
		{"carriage return", "aGk=\r", false},
		{"space", " aGk=", false},
		{"URL alphabet", "__8=", false},
		{"unicode", "éé", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateFormat("value", tc.value, Format("byte"))
			shape := &JSONShape{Kind: "string", Rules: JSONShapeRules{Formats: []Format{FormatBase64}}}
			wire, marshalErr := json.Marshal(tc.value)
			require.NoError(t, marshalErr)
			matched, matchErr := shape.Match(wire)
			require.NoError(t, matchErr)
			require.Equal(t, tc.valid, matched)
			if tc.valid {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			var validation *ServiceError
			require.True(t, errors.As(err, &validation), "must reject through the format contract, not unknown-format fallback: %v", err)
			require.Equal(t, "invalid_format", validation.Name)
		})
	}
}
