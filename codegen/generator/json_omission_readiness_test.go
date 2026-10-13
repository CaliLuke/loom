package generator

import "testing"

func TestJSONOmissionReadinessGenerated(t *testing.T) {
	runDesignHarness(t, "example.com/omission", func() {
		jsonTagNormalizationDSL(false)
	}, jsonTagNormalizationHarness)
}
