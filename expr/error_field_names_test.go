package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestErrorFieldNames(t *testing.T) {
	for _, field := range []string{"error", "loom_error_name", "loom_error_remedy"} {
		for _, shape := range []string{"inline", "named", "extended", "wrapper"} {
			for _, override := range []bool{false, true} {
				name := field + "/" + shape
				if override {
					name += "/override"
				}
				t.Run(name, func(t *testing.T) {
					design := func() {
						fields := func() {
							Attribute(field, String, func() {
								if override {
									Meta("struct:field:name", "FailureDetail")
								}
							})
						}
						var data expr.DataType
						var base expr.DataType
						switch shape {
						case "named":
							data = Type("Failure", fields)
						case "wrapper":
							base = Type("Base", fields)
							data = Type("Failure", func() {
								Attribute("detail", String)
							})
						case "extended":
							base := Type("Base", fields)
							data = Type("Failure", func() {
								Extend(base)
							})
						}
						Service("errors", func() {
							Method("show", func() {
								options := func() {
									if shape == "inline" {
										fields()
									}
									if shape == "wrapper" {
										Extend(base)
									}
									if field == "loom_error_remedy" {
										Remedy(func() {
											RemedyCode("fix")
										})
									}
								}
								if shape == "inline" {
									Error("failure", options)
								} else {
									Error("failure", data, options)
								}
							})
						})
					}
					if override {
						expr.RunDSL(t, design)
					} else {
						err := expr.RunInvalidDSL(t, design)
						require.ErrorContains(t, err, "conflicts with generated error method")
						require.ErrorContains(t, err, "struct:field:name")
					}
				})
			}
		}
	}
}

func TestErrorFieldNamesAllowed(t *testing.T) {
	expr.RunDSL(t, func() {
		ordinary := Type("Ordinary", func() {
			Attribute("error", String)
			Attribute("loom_error_name", String)
		})
		Service("errors", func() {
			Method("show", func() {
				Payload(ordinary)
				Error("failure", func() {
					Attribute("nested", ordinary)
					Attribute("loom_error_remedy", String)
				})
			})
		})
	})
}

func TestErrorFieldNameOverrides(t *testing.T) {
	for _, scope := range []string{"api", "service", "method"} {
		t.Run(scope, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				declare := func() {
					Error("failure", func() {
						Attribute("detail", String, func() {
							Meta("struct:field:name", "Error")
						})
					})
				}
				switch scope {
				case "api":
					API("errors", declare)
					Service("errors", func() {
						Method("show", func() {
							Error("failure")
						})
					})
				case "service":
					Service("errors", declare)
				case "method":
					Service("errors", func() {
						Method("show", declare)
					})
				}
			})
			require.ErrorContains(t, err, `attribute "detail" conflicts with generated error method "Error"`)
		})
	}
}
