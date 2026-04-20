package cli

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

var (
	boolN    = codegen.GoNativeTypeName(expr.Boolean)
	intN     = codegen.GoNativeTypeName(expr.Int)
	int32N   = codegen.GoNativeTypeName(expr.Int32)
	int64N   = codegen.GoNativeTypeName(expr.Int64)
	uintN    = codegen.GoNativeTypeName(expr.UInt)
	uint32N  = codegen.GoNativeTypeName(expr.UInt32)
	uint64N  = codegen.GoNativeTypeName(expr.UInt64)
	float32N = codegen.GoNativeTypeName(expr.Float32)
	float64N = codegen.GoNativeTypeName(expr.Float64)
	stringN  = codegen.GoNativeTypeName(expr.String)
	bytesN   = codegen.GoNativeTypeName(expr.Bytes)
)

// conversionCode produces the code that converts the string contained in the
// variable named from to the value stored in the variable "to" of type
// typeName. The second return value indicates whether the "err" variable must
// be declared prior to the conversion code being rendered. The last return
// value indicates whether the generated code can produce errors (i.e.
// initialize the err variable).
func conversionCode(from, to, typeName string, pointer bool) (*jen.Statement, bool, bool) {
	target, decl := conversionTarget(to, typeName, pointer)
	needCast := typeName != stringN && typeName != bytesN && flagType(typeName) != "JSON"
	parse, cast, declErr, checkErr := conversionStatements(from, target, typeName, pointer, decl)
	if !needCast {
		return parse, declErr, checkErr
	}
	if cast != nil {
		parse.Line().Add(cast)
	}
	if to != target {
		ref := ""
		if pointer {
			ref = "&"
		}
		parse.Line().Add(codegen.Expr(to + " = " + ref + target))
	}
	return parse, declErr, checkErr
}

func conversionTarget(to, typeName string, pointer bool) (target, decl string) {
	target = to
	if (typeName == stringN || typeName == bytesN || flagType(typeName) == "JSON") || !pointer {
		return target, ""
	}
	return "val", ":"
}

func conversionStatements(from, target, typeName string, pointer bool, decl string) (parse, cast *jen.Statement, declErr, checkErr bool) {
	declErr = true
	checkErr = true
	switch typeName {
	case boolN:
		parse = boolConversionParse(from, target, pointer)
	case intN:
		parse, cast = integerConversionParse(from, target, decl, "strconv.IntSize", "int")
	case int32N:
		parse, cast = integerConversionParse(from, target, decl, "32", "int32")
	case int64N:
		parse = integerDirectParseExpr("strconv.ParseInt", from, target, decl, "64")
		declErr = decl == ""
	case uintN:
		parse, cast = unsignedConversionParse(from, target, decl, "strconv.IntSize", "uint")
	case uint32N:
		parse, cast = unsignedConversionParse(from, target, decl, "32", "uint32")
	case uint64N:
		parse = integerDirectParseExpr("strconv.ParseUint", from, target, decl, "64")
		declErr = decl == ""
	case float32N:
		parse, cast = floatConversionParse(from, target, decl, "32", "float32")
	case float64N:
		parse = floatDirectParseExpr(from, target, decl, "64")
		declErr = decl == ""
	case stringN:
		parse = codegen.Expr(target + " " + decl + "= " + from)
		declErr = false
		checkErr = false
	case bytesN:
		parse = codegen.Expr(target + " " + decl + "= []byte(" + from + ")")
		declErr = false
		checkErr = false
	default:
		parse = codegen.Expr("err = json.Unmarshal([]byte(" + from + "), &" + target + ")")
	}
	return parse, cast, declErr, checkErr
}

func boolConversionParse(from, target string, pointer bool) *jen.Statement {
	parse := new(jen.Statement)
	if pointer {
		parse = codegen.Expr("var " + target + " bool\n")
	}
	return parse.Add(codegen.Expr(target + ", err = strconv.ParseBool(" + from + ")"))
}

func integerConversionParse(from, target, decl, bits, castType string) (*jen.Statement, *jen.Statement) {
	parse := codegen.Expr("var v int64\nv, err = strconv.ParseInt(" + from + ", 10, " + bits + ")")
	cast := codegen.Expr(target + " " + decl + "= " + castType + "(v)")
	return parse, cast
}

func unsignedConversionParse(from, target, decl, bits, castType string) (*jen.Statement, *jen.Statement) {
	parse := codegen.Expr("var v uint64\nv, err = strconv.ParseUint(" + from + ", 10, " + bits + ")")
	cast := codegen.Expr(target + " " + decl + "= " + castType + "(v)")
	return parse, cast
}

func floatConversionParse(from, target, decl, bits, castType string) (*jen.Statement, *jen.Statement) {
	parse := codegen.Expr("var v float64\nv, err = strconv.ParseFloat(" + from + ", " + bits + ")")
	cast := codegen.Expr(target + " " + decl + "= " + castType + "(v)")
	return parse, cast
}

func integerDirectParseExpr(parseFn, from, target, decl, bits string) *jen.Statement {
	return codegen.Expr(target + ", err " + decl + "= " + parseFn + "(" + from + ", 10, " + bits + ")")
}

func floatDirectParseExpr(from, target, decl, bits string) *jen.Statement {
	return codegen.Expr(target + ", err " + decl + "= strconv.ParseFloat(" + from + ", " + bits + ")")
}

// flagType calculates the type of a flag
func flagType(tname string) string {
	switch tname {
	case boolN, intN, int32N, int64N, uintN, uint32N, uint64N, float32N, float64N, stringN:
		return strings.ToUpper(tname)
	case bytesN:
		return "STRING"
	default: // Any, Array, Map, Object, User
		return "JSON"
	}
}

// jsonExample generates a json example
func jsonExample(v any) string {
	// In JSON, keys must be a string. But Loom allows map keys to be anything.
	r := reflect.ValueOf(v)
	if r.Kind() == reflect.Map {
		keys := r.MapKeys()
		if len(keys) == 0 {
			b, err := json.MarshalIndent(v, "   ", "   ")
			if err == nil {
				return string(b)
			}
			return "{}"
		}
		if keys[0].Kind() != reflect.String {
			a := make(map[string]any, len(keys))
			var kstr string
			for _, k := range keys {
				switch t := k.Interface().(type) {
				case bool:
					kstr = strconv.FormatBool(t)
				case int32:
					kstr = strconv.FormatInt(int64(t), 10)
				case int64:
					kstr = strconv.FormatInt(t, 10)
				case int:
					kstr = strconv.Itoa(t)
				case float32:
					kstr = strconv.FormatFloat(float64(t), 'f', -1, 32)
				case float64:
					kstr = strconv.FormatFloat(t, 'f', -1, 64)
				default:
					kstr = k.String()
				}
				a[kstr] = r.MapIndex(k).Interface()
			}
			v = a
		}
	}
	b, err := json.MarshalIndent(v, "   ", "   ")
	ex := "?"
	if err == nil {
		ex = string(b)
	}
	if strings.Contains(ex, "\n") {
		ex = "'" + strings.ReplaceAll(ex, "'", "\\'") + "'"
	}
	return ex
}
