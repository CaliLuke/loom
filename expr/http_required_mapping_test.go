package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestRequiredMappingReadiness(t *testing.T) {
	for _, shape := range []string{"selected request", "listed request", "listed error"} {
		t.Run(shape, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				failure := Type("Failure", func() {
					ErrorName("kind", String)
					Attribute("detail", String)
					Required("kind", "detail")
				})
				Service("mapping", func() {
					Method("send", func() {
						Payload(func() {
							Attribute("lost", String)
							Attribute("sent", String)
							if shape != "listed error" {
								Required("lost")
							}
						})
						Error("busy", failure)
						HTTP(func() {
							POST("/send")
							if shape == "selected request" {
								Body("sent")
							} else if shape == "listed request" {
								Body(func() {
									Attribute("sent")
								})
							}
							if shape == "listed error" {
								Response("busy", StatusBadRequest, func() {
									Body(func() {
										Attribute("detail")
									})
								})
							}
						})
					})
				})
			})
			missing := "lost"
			if shape == "listed error" {
				missing = "kind"
			}
			require.ErrorContains(t, err, missing)
			require.ErrorContains(t, err, "no reconstruction source")
		})
	}
}

func TestRequiredMappingControls(t *testing.T) {
	for _, location := range []string{"header", "query", "cookie", "optional omission"} {
		t.Run(location, func(t *testing.T) {
			expr.RunDSL(t, func() {
				Service("mapping", func() {
					Method("send", func() {
						Payload(func() {
							Attribute("mapped", String)
							Attribute("sent", String)
							if location != "optional omission" {
								Required("mapped")
							}
						})
						HTTP(func() {
							POST("/send")
							Body("sent")
							switch location {
							case "header":
								Header("mapped:x-value")
							case "query":
								Param("mapped:value")
							case "cookie":
								Cookie("mapped:value")
							}
						})
					})
				})
			})
		})
	}
}
