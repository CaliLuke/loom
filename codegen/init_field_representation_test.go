package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestInitStructFieldsPhysicalPrimitive(t *testing.T) {
	for _, replacement := range []string{"", "int64", "int32"} {
		for _, pointer := range []bool{false, true} {
			name := replacement
			if name == "" {
				name = "plain"
			}
			if pointer {
				name += "/pointer"
			}
			t.Run(name, func(t *testing.T) {
				source := &expr.AttributeExpr{Type: expr.Int}
				target := &expr.AttributeExpr{Type: expr.Int}
				if replacement != "" {
					target.Meta = expr.MetaExpr{"struct:field:type": {replacement}}
				}
				plan, err := InitStructFields([]*InitArgData{{Name: "status", Type: expr.Int, Attribute: source, FieldName: "Status", FieldType: expr.Int, FieldAttribute: target, FieldPointer: pointer}}, "v", "", "service", NewNameScope())
				require.NoError(t, err)
				switch {
				case replacement != "":
					expected := "v.Status = int64(status)\n"
					if pointer {
						expected = "tmpstatus := int64(status)\nv.Status = &tmpstatus\n"
					}
					if replacement == "int32" {
						assignment := "v.Status = convertedValue\n"
						if pointer {
							assignment = "tmpstatus := convertedValue\nv.Status = &tmpstatus\n"
						}
						expected = "if *transformErr == nil {\nconvertedValue := int32(status)\nif int(convertedValue) != status || (status < 0) != (convertedValue < 0) {\n*transformErr = fmt.Errorf(\"value %v cannot be represented as int32\", status)\n} else {\n" + assignment + "}\n}\n"
					}
					require.Equal(t, expected, plan.Code)
				case pointer:
					require.Equal(t, "v.Status = &status\n", plan.Code)
				default:
					require.Equal(t, "v.Status = status\n", plan.Code)
				}
				if pointer && replacement != "" {
					require.Contains(t, plan.Code, "v.Status = &tmpstatus")
				}
			})
		}
	}
	t.Run("named", func(t *testing.T) {
		target := &expr.UserTypeExpr{TypeName: "Status", UID: "status", AttributeExpr: &expr.AttributeExpr{Type: expr.Int}}
		plan, err := InitStructFields([]*InitArgData{{Name: "status", Type: expr.Int, FieldName: "Status", FieldType: target, FieldAttribute: &expr.AttributeExpr{Type: target}}}, "v", "", "service", NewNameScope())
		require.NoError(t, err)
		require.Equal(t, "v.Status = service.Status(status)\n", plan.Code)
	})
}

func TestInitStructFieldsRejectsLossyInteger(t *testing.T) {
	target := &expr.AttributeExpr{Type: expr.Int, Meta: expr.MetaExpr{"struct:field:type": {"int8"}}}
	plan, err := InitStructFields([]*InitArgData{{Name: "status", Type: expr.Int, FieldName: "Status", FieldType: expr.Int, FieldAttribute: target}}, "v", "", "service", NewNameScope())
	require.NoError(t, err)
	require.True(t, plan.ErrorAware, "native int8 must not silently truncate wire status 200")
}
