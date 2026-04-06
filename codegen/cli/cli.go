// Package cli contains helpers used by transport-specific command-line client
// generators for parsing the command-line flags to identify the service and
// the method to make a request along with the request payload to be sent.
package cli

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
)

type (
	// CommandData contains the data needed to render a command.
	CommandData struct {
		// Name of command e.g. "cellar-storage"
		Name string
		// VarName is the name of the command variable e.g.
		// "cellarStorage"
		VarName string
		// Description is the help text.
		Description string
		// Subcommands is the list of endpoint commands.
		Subcommands []*SubcommandData
		// Example is a valid command invocation, starting with the
		// command name.
		Example string
		// PkgName is the service HTTP client package import name,
		// e.g. "storagec".
		PkgName string
		// Interceptors contains the data for client interceptors if any.
		Interceptors *InterceptorData
	}

	// SubcommandData contains the data needed to render a sub-command.
	SubcommandData struct {
		// Name is the sub-command name e.g. "add"
		Name string
		// FullName is the sub-command full name e.g. "storageAdd"
		FullName string
		// Description is the help text.
		Description string
		// Flags is the list of flags supported by the subcommand.
		Flags []*FlagData
		// MethodVarName is the endpoint method name, e.g. "Add"
		MethodVarName string
		// BuildFunction contains the data to generate a payload builder function
		// if any. Exclusive with Conversion.
		BuildFunction *BuildFunctionData
		// Conversion contains the flag value to payload conversion function if
		// any. Exclusive with BuildFunction.
		Conversion *jen.Statement
		// Example is a valid command invocation, starting with the command name.
		Example string
		// Interceptors contains the data for client interceptors if any apply to the endpoint method.
		Interceptors *InterceptorData
	}

	// InterceptorData contains the data needed to generate interceptor code.
	InterceptorData struct {
		// VarName is the name of the interceptor variable.
		VarName string
		// PkgName is the package name containing the interceptor type.
		PkgName string
	}

	// FlagData contains the data needed to render a command-line flag.
	FlagData struct {
		// Name is the name of the flag, e.g. "list-vintage"
		Name string
		// VarName is the name of the flag variable, e.g. "listVintage"
		VarName string
		// Type is the type of the flag, e.g. INT
		Type string
		// FullName is the flag full name e.g. "storageAddVintage"
		FullName string
		// Description is the flag help text.
		Description string
		// Required is true if the flag is required.
		Required bool
		// Example returns a JSON serialized example value.
		Example string
		// Default returns the default value if any.
		Default any
	}

	// BuildFunctionData contains the data needed to generate a constructor
	// function that builds a service method payload type from the command-line
	// flags.
	BuildFunctionData struct {
		// Name is the build payload function name.
		Name string
		// Description describes the payload function.
		Description string
		// ActualParams is the list of passed build function parameters.
		ActualParams []string
		// FormalParams is the list of build function formal parameter
		// names.
		FormalParams []string
		// ServiceName is the name of the service.
		ServiceName string
		// MethodName is the name of the method.
		MethodName string
		// ResultType is the fully qualified payload type name.
		ResultType string
		// Fields describes the payload fields.
		Fields []*FieldData
		// PayloadInit contains the data needed to render the function
		// body.
		PayloadInit *PayloadInitData
		// CheckErr is true if the payload initialization code requires an
		// "err error" variable that must be checked.
		CheckErr bool
	}

	// FieldData contains the data needed to generate the code that initializes a
	// field in the method payload type.
	FieldData struct {
		// Name is the field name, e.g. "Vintage"
		Name string
		// VarName is the name of the local variable holding the field
		// value, e.g. "vintage"
		VarName string
		// TypeRef is the reference to the type.
		TypeRef string
		// Init is the code initializing the variable.
		Init *jen.Statement
	}

	// PayloadInitData contains the data needed to generate a constructor
	// function that initializes a service method payload type from the
	// command-ling arguments.
	PayloadInitData struct {
		// Code is the payload initialization code.
		Code *jen.Statement
		// ReturnTypeAttribute if non-empty returns an attribute in the payload
		// type that describes the shape of the method payload.
		ReturnTypeAttribute string
		// ReturnTypeAttributePointer is true if the return type attribute
		// generated struct field holds a pointer
		ReturnTypeAttributePointer bool
		// ReturnIsStruct if true indicates that the method payload is an object.
		ReturnIsStruct bool
		// ReturnTypeName is the fully-qualified name of the payload.
		ReturnTypeName string
		// ReturnTypePkg is the package name where the payload is present.
		ReturnTypePkg string
		// Args is the list of arguments for the constructor.
		Args []*codegen.InitArgData
	}
)

// BuildCommandData builds the data needed by CLI code generators to render the
// parsing of the service command.
func BuildCommandData(data *service.Data) *CommandData {
	description := data.Description
	if description == "" {
		description = fmt.Sprintf("Make requests to the %q service", data.Name)
	}

	var interceptors *InterceptorData
	if len(data.ClientInterceptors) > 0 {
		interceptors = &InterceptorData{
			VarName: codegen.Goify(data.Name, false) + "Inter",
			PkgName: data.PkgName,
		}
	}

	return &CommandData{
		Name:         codegen.KebabCase(data.Name),
		VarName:      codegen.Goify(data.Name, false),
		Description:  description,
		PkgName:      data.PkgName + "c",
		Interceptors: interceptors,
	}
}

// BuildSubcommandData builds the data needed by CLI code generators to render
// the CLI parsing of the service sub-command.
func BuildSubcommandData(data *service.Data, m *service.MethodData, buildFunction *BuildFunctionData, flags []*FlagData) *SubcommandData {
	en := m.Name
	name := codegen.KebabCase(en)
	fullName := goifyTerms(data.Name, en)
	description := m.Description
	if description == "" {
		description = fmt.Sprintf("Make request to the %q endpoint", m.Name)
	}

	var conversion *jen.Statement
	if m.Payload != "" && buildFunction == nil && len(flags) > 0 {
		// No build function, just convert the arg to the body type
		var convPre, convSuff string
		target := "data"
		if flagType(m.Payload) == "JSON" {
			target = "val"
			convPre = "var val " + m.Payload + "\n"
			convSuff = "\ndata = val"
		}
		conv, _, check := conversionCode(
			"*"+flags[0].FullName+"Flag",
			target,
			m.Payload,
			false,
		)
		conversion = codegen.Expr(convPre).Add(conv).Add(codegen.Expr(convSuff))
		if check {
			conversion = codegen.Expr("var err error\n").Add(conversion).Line()
			conversion.If(jen.Err().Op("!=").Nil()).BlockFunc(func(group *jen.Group) {
				var ret *jen.Statement
				if flagType(m.Payload) == "JSON" {
					ret = jen.Return(
						jen.Nil(),
						jen.Nil(),
						jen.Qual("fmt", "Errorf").Call(
							jen.Lit("invalid JSON for "+flags[0].FullName+"Flag, \nerror: %s, \nexample of valid JSON:\n%s"),
							jen.Err(),
							jen.Lit(flags[0].Example),
						),
					)
				} else {
					ret = jen.Return(
						jen.Nil(),
						jen.Nil(),
						jen.Qual("fmt", "Errorf").Call(
							jen.Lit("invalid value for "+flags[0].FullName+"Flag, must be "+flags[0].Type),
						),
					)
				}
				group.Add(ret)
			})
		}
	}

	var interceptors *InterceptorData
	if len(m.ClientInterceptors) > 0 {
		interceptors = &InterceptorData{
			VarName: codegen.Goify(data.Name, false) + "Inter",
			PkgName: data.PkgName,
		}
	}
	sub := &SubcommandData{
		Name:          name,
		FullName:      fullName,
		Description:   description,
		Flags:         flags,
		MethodVarName: m.VarName,
		BuildFunction: buildFunction,
		Conversion:    conversion,
		Interceptors:  interceptors,
	}
	generateExample(sub, data.Name)

	return sub
}

// UsageCommands builds a section that generates a help text showing
// the list of allowed commands and sub-commands.
func UsageCommands(data []*CommandData) codegen.Section {
	usages := make([]string, len(data))
	for i, cmd := range data {
		subs := make([]string, len(cmd.Subcommands))
		for i, s := range cmd.Subcommands {
			subs[i] = s.Name
		}
		var lp, rp string
		if len(subs) > 1 {
			lp = "("
			rp = ")"
		}
		usages[i] = fmt.Sprintf("%s %s%s%s", cmd.Name, lp, strings.Join(subs, "|"), rp)
	}

	return codegen.MustJenniferSection("cli-usage-commands", func(stmt *jen.Statement) {
		stmt.Comment("UsageCommands returns the set of commands and sub-commands using the format").Line()
		stmt.Comment("").Line()
		stmt.Comment("   command (subcommand1|subcommand2|...)").Line()
		stmt.Func().Id("UsageCommands").Params().Index().String().BlockFunc(func(group *jen.Group) {
			group.Return().Index().String().ValuesFunc(func(values *jen.Group) {
				for _, usage := range usages {
					values.Lit(usage)
				}
			})
		})
	})
}

// UsageExamples builds a section that generates a help text showing
// a valid invocation of the CLI tool.
func UsageExamples(data []*CommandData) codegen.Section {
	var examples []string
	for i, cmd := range data {
		if i < 5 {
			examples = append(examples, cmd.Example)
		}
	}

	return codegen.MustJenniferSection("cli-usage-examples", func(stmt *jen.Statement) {
		stmt.Comment("UsageExamples produces an example of a valid invocation of the CLI tool.").Line()
		stmt.Func().Id("UsageExamples").Params().String().BlockFunc(func(group *jen.Group) {
			if len(examples) == 0 {
				group.Return(jen.Lit(""))
				return
			}
			var expr *jen.Statement
			for i, example := range examples {
				part := jen.Id("os").Dot("Args").Index(jen.Lit(0)).Op("+").Lit(" " + example + "\\n")
				if i == 0 {
					expr = part
					continue
				}
				expr.Op("+").Add(part)
			}
			group.Return(expr)
		})
	})
}

// FlagsCode returns a string containing the code that parses the command-line
// flags to infer the command (service), sub-command (method), and the
// arguments (method payload) invoked by the tool. It panics if any error
// occurs during the generation of flag parsing code.
func FlagsCode(data []*CommandData) string {
	return renderJenniferStatement(FlagsCodeStatement(data))
}

// CommandUsage builds the section that can be used to generate the
// endpoint command usage code.
func CommandUsage(data *CommandData) codegen.Section {
	return codegen.MustJenniferSection("cli-command-usage", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%sUsage displays the usage of the %s command and its subcommands.", data.VarName, data.Name))
		stmt.Func().Id(data.VarName + "Usage").Params().BlockFunc(func(group *jen.Group) {
			group.Qual("fmt", "Fprintln").Call(jen.Qual("os", "Stderr"), jen.Lit(printDescription(data.Description)))
			group.Qual("fmt", "Fprintf").Call(jen.Qual("os", "Stderr"), jen.Lit("Usage:\n    %s [globalflags] "+data.Name+" COMMAND [flags]\n\n"), jen.Qual("os", "Args").Index(jen.Lit(0)))
			group.Qual("fmt", "Fprintln").Call(jen.Qual("os", "Stderr"), jen.Lit("COMMAND:"))
			for _, sub := range data.Subcommands {
				group.Qual("fmt", "Fprintln").Call(jen.Qual("os", "Stderr"), jen.Lit("    "+sub.Name+": "+printDescription(sub.Description)))
			}
			group.Qual("fmt", "Fprintln").Call(jen.Qual("os", "Stderr"))
			group.Qual("fmt", "Fprintln").Call(jen.Qual("os", "Stderr"), jen.Lit("Additional help:"))
			group.Qual("fmt", "Fprintf").Call(jen.Qual("os", "Stderr"), jen.Lit("    %s "+data.Name+" COMMAND --help\n"), jen.Qual("os", "Args").Index(jen.Lit(0)))
		})
		stmt.Line()
		for _, sub := range data.Subcommands {
			stmt.Func().Id(sub.FullName + "Usage").Params().BlockFunc(func(group *jen.Group) {
				group.Comment("Header with flags")
				group.Qual("fmt", "Fprintf").Call(jen.Qual("os", "Stderr"), jen.Lit("%s [flags] "+data.Name+" "+sub.Name), jen.Qual("os", "Args").Index(jen.Lit(0)))
				for _, flag := range sub.Flags {
					group.Qual("fmt", "Fprint").Call(jen.Qual("os", "Stderr"), jen.Lit(" -"+flag.Name+" "+flag.Type))
				}
				group.Qual("fmt", "Fprintln").Call(jen.Qual("os", "Stderr"))
				group.Line()
				group.Comment("Description")
				group.Qual("fmt", "Fprintln").Call(jen.Qual("os", "Stderr"))
				group.Qual("fmt", "Fprintln").Call(jen.Qual("os", "Stderr"), jen.Lit(printDescription(sub.Description)))
				group.Line()
				group.Comment("Flags list")
				for _, flag := range sub.Flags {
					group.Qual("fmt", "Fprintln").Call(jen.Qual("os", "Stderr"), jen.Lit("    -"+flag.Name+" "+flag.Type+": "+flag.Description))
				}
				group.Line()
				group.Qual("fmt", "Fprintln").Call(jen.Qual("os", "Stderr"))
				group.Qual("fmt", "Fprintln").Call(jen.Qual("os", "Stderr"), jen.Lit("Example:"))
				group.Qual("fmt", "Fprintf").Call(jen.Qual("os", "Stderr"), jen.Lit("    %s %s\n"), jen.Qual("os", "Args").Index(jen.Lit(0)), jen.Lit(sub.Example))
			})
			stmt.Line()
		}
	})
}

// PayloadBuilderSection builds the section that can be used to
// generate the payload builder code.
func PayloadBuilderSection(buildFunction *BuildFunctionData) codegen.Section {
	return codegen.MustJenniferSection("cli-build-payload", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s builds the payload for the %s %s endpoint from CLI flags.", buildFunction.Name, buildFunction.ServiceName, buildFunction.MethodName))
		fn := stmt.Func().Id(buildFunction.Name).ParamsFunc(func(group *jen.Group) {
			for _, formal := range buildFunction.FormalParams {
				group.Id(formal).String()
			}
		}).Params(codegen.TypeRef(buildFunction.ResultType), jen.Error())
		fn.BlockFunc(func(group *jen.Group) {
			if buildFunction.CheckErr {
				group.Var().Err().Error()
			}
			for _, field := range buildFunction.Fields {
				if field.VarName == "" {
					continue
				}
				group.Var().Id(field.VarName).Add(codegen.TypeRef(field.TypeRef))
				group.Block(field.Init)
			}
			if buildFunction.PayloadInit != nil {
				if buildFunction.PayloadInit.Code != nil {
					group.Add(buildFunction.PayloadInit.Code)
					if buildFunction.PayloadInit.ReturnTypeAttribute != "" {
						group.Id("res").Op(":=").Op("&").Add(codegen.TypeRef(buildFunction.PayloadInit.ReturnTypeName)).ValuesFunc(func(values *jen.Group) {
							value := values.Id(buildFunction.PayloadInit.ReturnTypeAttribute).Op(":")
							if buildFunction.PayloadInit.ReturnTypeAttributePointer {
								value.Op("&")
							}
							value.Id("v")
						})
					}
				}
				if buildFunction.PayloadInit.ReturnIsStruct {
					if buildFunction.PayloadInit.Code == nil {
						target := "v"
						if buildFunction.PayloadInit.ReturnTypeAttribute != "" {
							target = "res"
						}
						group.Id(target).Op(":=").Op("&").Add(codegen.TypeRef(buildFunction.PayloadInit.ReturnTypeName)).Values()
					}
					group.Add(fieldCode(buildFunction.PayloadInit))
				}
				resultVar := "v"
				if buildFunction.PayloadInit.ReturnTypeAttribute != "" {
					resultVar = "res"
				}
				group.Return(codegen.Expr(resultVar), jen.Nil())
			}
		})
	})
}

// NewFlagData creates a new FlagData from the given argument attributes.
//
// svcn is the service name
// en is the endpoint name
// name is the flag name
// typeName is the flag type
// description is the flag description
// required determines if the flag is required
// example is an example value for the flag
func NewFlagData(svcn, en, name, typeName, description string, required bool, example, def any) *FlagData {
	ex := jsonExample(example)
	fn := goifyTerms(svcn, en, name)
	return &FlagData{
		Name:        codegen.KebabCase(name),
		VarName:     codegen.Goify(name, false),
		Type:        flagType(typeName),
		FullName:    fn,
		Description: description,
		Required:    required,
		Example:     ex,
		Default:     def,
	}
}

// FieldLoadCode returns the code used in the build payload function that
// initializes one of the payload object fields. It returns the initialization
// code and a boolean indicating whether the code requires an "err" variable.
func FieldLoadCode(f *FlagData, argName, argTypeName, validate string, defaultValue any, payload expr.DataType, payloadRef string) (*jen.Statement, bool) {
	var (
		code    *jen.Statement
		declErr bool
	)
	if argTypeName == codegen.GoNativeTypeName(expr.String) {
		ref := "&"
		if f.Required || defaultValue != nil {
			ref = ""
		}
		code = codegen.Expr(argName + " = " + ref + f.FullName)
		declErr = validate != ""
	} else {
		var checkErr bool
		code, declErr, checkErr = conversionCode(f.FullName, argName, argTypeName, !f.Required && defaultValue == nil)
		if checkErr {
			code.Line().If(jen.Err().Op("!=").Nil()).BlockFunc(func(group *jen.Group) {
				nilVal := "nil"
				if expr.IsPrimitive(payload) {
					group.Add(codegen.Expr("var zero " + payloadRef))
					nilVal = "zero"
				}
				if flagType(argTypeName) == "JSON" {
					group.Return(
						codegen.Expr(nilVal),
						jen.Qual("fmt", "Errorf").Call(
							jen.Lit("invalid JSON for "+argName+", \nerror: %s, \nexample of valid JSON:\n%s"),
							jen.Err(),
							jen.Lit(f.Example),
						),
					)
				} else {
					group.Return(
						codegen.Expr(nilVal),
						jen.Qual("fmt", "Errorf").Call(
							jen.Lit("invalid value for "+argName+", must be "+f.Type),
						),
					)
				}
			})
		}
	}
	if validate != "" {
		nilCheck := "if " + argName + " != nil {"
		if strings.HasPrefix(validate, nilCheck) {
			// hackety hack... the validation code is generated for the client and needs to
			// account for the fact that the field could be nil in this case. We are reusing
			// that code to validate a CLI flag which can never be nil.  Lint tools complain
			// about that so remove the if statements. Ideally we'd have a better way to do
			// this but that requires a lot of changes and the added complexity might not be
			// worth it.
			var lines []string
			ls := strings.Split(validate, "\n")
			for i := 1; i < len(ls)-1; i++ {
				if ls[i+1] == nilCheck {
					i++ // skip both closing brace on previous line and check
					continue
				}
				lines = append(lines, ls[i])
			}
			validate = strings.Join(lines, "\n")
		}
		code.Line().Add(codegen.Expr(validate)).Line()
		nilVal := "nil"
		if expr.IsPrimitive(payload) {
			code.Add(codegen.Expr("var zero " + payloadRef)).Line()
			nilVal = "zero"
		}
		code.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(codegen.Expr(nilVal), jen.Err()),
		)
	}
	if !f.Required {
		return jen.If(codegen.Expr(f.FullName).Op("!=").Lit("")).Block(code), declErr
	}
	return code, declErr
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
	var (
		parse *jen.Statement
		cast  *jen.Statement

		target   = to
		needCast = typeName != stringN && typeName != bytesN && flagType(typeName) != "JSON"
		declErr  = true
		checkErr = true
		decl     = ""
	)
	if needCast && pointer {
		target = "val"
		decl = ":"
	}
	switch typeName {
	case boolN:
		parse = new(jen.Statement)
		if pointer {
			parse = codegen.Expr("var " + target + " bool\n")
		}
		parse.Add(codegen.Expr(target + ", err = strconv.ParseBool(" + from + ")"))
	case intN:
		parse = codegen.Expr("var v int64\nv, err = strconv.ParseInt(" + from + ", 10, strconv.IntSize)")
		cast = codegen.Expr(target + " " + decl + "= int(v)")
	case int32N:
		parse = codegen.Expr("var v int64\nv, err = strconv.ParseInt(" + from + ", 10, 32)")
		cast = codegen.Expr(target + " " + decl + "= int32(v)")
	case int64N:
		parse = codegen.Expr(target + ", err " + decl + "= strconv.ParseInt(" + from + ", 10, 64)")
		declErr = decl == ""
	case uintN:
		parse = codegen.Expr("var v uint64\nv, err = strconv.ParseUint(" + from + ", 10, strconv.IntSize)")
		cast = codegen.Expr(target + " " + decl + "= uint(v)")
	case uint32N:
		parse = codegen.Expr("var v uint64\nv, err = strconv.ParseUint(" + from + ", 10, 32)")
		cast = codegen.Expr(target + " " + decl + "= uint32(v)")
	case uint64N:
		parse = codegen.Expr(target + ", err " + decl + "= strconv.ParseUint(" + from + ", 10, 64)")
		declErr = decl == ""
	case float32N:
		parse = codegen.Expr("var v float64\nv, err = strconv.ParseFloat(" + from + ", 32)")
		cast = codegen.Expr(target + " " + decl + "= float32(v)")
	case float64N:
		parse = codegen.Expr(target + ", err " + decl + "= strconv.ParseFloat(" + from + ", 64)")
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

// goifyTerms makes valid go identifiers out of the supplied terms
func goifyTerms(terms ...string) string {
	res := codegen.Goify(terms[0], false)
	if len(terms) == 1 {
		return res
	}
	for _, t := range terms[1:] {
		res += codegen.Goify(t, true)
	}
	return res
}

func printDescription(desc string) string {
	res := strings.ReplaceAll(desc, "`", "`+\"`\"+`")
	res = strings.ReplaceAll(res, "\n", "\n\t")
	return res
}

func generateExample(sub *SubcommandData, svc string) {
	ex := codegen.KebabCase(svc) + " " + codegen.KebabCase(sub.Name)
	for _, f := range sub.Flags {
		ex += " --" + f.Name + " " + f.Example
	}
	sub.Example = ex
}

// fieldCode generates code to initialize the data structures fields
// from the given args. It is used only in templates.
func fieldCode(init *PayloadInitData) *jen.Statement {
	varn := "res"
	if init.ReturnTypeAttribute == "" {
		varn = "v"
	}
	// We can ignore the transform helpers as there won't be any generated
	// because the args cannot be user types.
	c, _, err := codegen.InitStructFields(init.Args, varn, "", init.ReturnTypePkg)
	if err != nil {
		panic(err) // bug
	}
	return codegen.Expr(c)
}

// FlagsCodeStatement builds the CLI flag parsing statements.
func FlagsCodeStatement(data []*CommandData) *jen.Statement {
	stmt := new(jen.Statement)
	appendFlagDefinitions(stmt, data)
	appendUsageAssignments(stmt, data)
	appendTopLevelParsing(stmt)
	appendServiceSelection(stmt, data)
	appendEndpointSelection(stmt, data)
	appendEndpointFlagParsing(stmt)
	stmt.Line()
	return stmt
}

func appendFlagDefinitions(stmt *jen.Statement, data []*CommandData) {
	stmt.Var().DefsFunc(func(group *jen.Group) {
		for _, cmd := range data {
			group.Id(cmd.VarName+"Flags").Op("=").Qual("flag", "NewFlagSet").Call(jen.Lit(cmd.Name), jen.Qual("flag", "ContinueOnError"))
			appendSubcommandFlagDefinitions(group, cmd.Subcommands)
		}
	}).Line()
}

func appendSubcommandFlagDefinitions(group *jen.Group, subcommands []*SubcommandData) {
	for _, sub := range subcommands {
		group.Id(sub.FullName+"Flags").Op("=").Qual("flag", "NewFlagSet").Call(jen.Lit(sub.Name), jen.Qual("flag", "ExitOnError"))
		for _, flag := range sub.Flags {
			group.Id(flag.FullName+"Flag").Op("=").Id(sub.FullName+"Flags").Dot("String").Call(
				jen.Lit(flag.Name),
				jen.Lit(flagDefaultValue(flag)),
				jen.Lit(flag.Description),
			)
		}
	}
}

func flagDefaultValue(flag *FlagData) string {
	if flag.Default != nil {
		return fmt.Sprint(flag.Default)
	}
	if flag.Required {
		return "REQUIRED"
	}
	return ""
}

func appendUsageAssignments(stmt *jen.Statement, data []*CommandData) {
	for _, cmd := range data {
		stmt.Id(cmd.VarName + "Flags").Dot("Usage").Op("=").Id(cmd.VarName + "Usage")
		stmt.Line()
		for _, sub := range cmd.Subcommands {
			stmt.Id(sub.FullName + "Flags").Dot("Usage").Op("=").Id(sub.FullName + "Usage")
			stmt.Line()
		}
	}
	stmt.Line()
}

func appendTopLevelParsing(stmt *jen.Statement) {
	stmt.If(
		jen.Err().Op(":=").Qual("flag", "CommandLine").Dot("Parse").Call(jen.Qual("os", "Args").Index(jen.Lit(1), jen.Empty())),
		jen.Err().Op("!=").Nil(),
	).Block(
		jen.Return(jen.Nil(), jen.Nil(), jen.Err()),
	)
	stmt.Line()
	stmt.If(jen.Qual("flag", "NArg").Call().Op("<").Lit(2)).Block(
		jen.Return(jen.Nil(), jen.Nil(), jen.Qual("fmt", "Errorf").Call(jen.Lit("not enough arguments"))),
	)
	stmt.Line()
}

func appendServiceSelection(stmt *jen.Statement, data []*CommandData) {
	stmt.Var().Defs(
		jen.Id("svcn").String(),
		jen.Id("svcf").Op("*").Qual("flag", "FlagSet"),
	)
	stmt.Line()
	stmt.Block(
		jen.Id("svcn").Op("=").Qual("flag", "Arg").Call(jen.Lit(0)),
		serviceSelectionSwitch(data),
	)
	stmt.Line()
	stmt.If(
		jen.Err().Op(":=").Id("svcf").Dot("Parse").Call(codegen.Expr("flag.Args()[1:]")),
		jen.Err().Op("!=").Nil(),
	).Block(
		jen.Return(jen.Nil(), jen.Nil(), jen.Err()),
	)
	stmt.Line()
}

func serviceSelectionSwitch(data []*CommandData) *jen.Statement {
	return jen.Switch(jen.Id("svcn")).BlockFunc(func(group *jen.Group) {
		for _, cmd := range data {
			group.Case(jen.Lit(cmd.Name)).Block(
				jen.Id("svcf").Op("=").Id(cmd.VarName + "Flags"),
			)
		}
		group.Default().Block(
			jen.Return(jen.Nil(), jen.Nil(), jen.Qual("fmt", "Errorf").Call(jen.Lit("unknown service %q"), jen.Id("svcn"))),
		)
	})
}

func appendEndpointSelection(stmt *jen.Statement, data []*CommandData) {
	stmt.Var().Defs(
		jen.Id("epn").String(),
		jen.Id("epf").Op("*").Qual("flag", "FlagSet"),
	)
	stmt.Line()
	stmt.Block(
		jen.Id("epn").Op("=").Id("svcf").Dot("Arg").Call(jen.Lit(0)),
		endpointSelectionSwitch(data),
	)
	stmt.Line()
	stmt.If(jen.Id("epf").Op("==").Nil()).Block(
		jen.Return(jen.Nil(), jen.Nil(), jen.Qual("fmt", "Errorf").Call(jen.Lit("unknown %q endpoint %q"), jen.Id("svcn"), jen.Id("epn"))),
	)
	stmt.Line()
}

func endpointSelectionSwitch(data []*CommandData) *jen.Statement {
	return jen.Switch(jen.Id("svcn")).BlockFunc(func(group *jen.Group) {
		for _, cmd := range data {
			group.Case(jen.Lit(cmd.Name)).Block(
				subcommandSelectionSwitch(cmd.Subcommands),
			)
		}
	})
}

func subcommandSelectionSwitch(subcommands []*SubcommandData) *jen.Statement {
	return jen.Switch(jen.Id("epn")).BlockFunc(func(group *jen.Group) {
		for _, sub := range subcommands {
			group.Case(jen.Lit(sub.Name)).Block(
				jen.Id("epf").Op("=").Id(sub.FullName + "Flags"),
			)
		}
	})
}

func appendEndpointFlagParsing(stmt *jen.Statement) {
	stmt.If(jen.Id("svcf").Dot("NArg").Call().Op(">").Lit(1)).Block(
		jen.If(
			jen.Err().Op(":=").Id("epf").Dot("Parse").Call(jen.Id("svcf").Dot("Args").Call().Index(jen.Lit(1), jen.Empty())),
			jen.Err().Op("!=").Nil(),
		).Block(
			jen.Return(jen.Nil(), jen.Nil(), jen.Err()),
		),
	)
}

func renderJenniferStatement(stmt *jen.Statement) string {
	file := jen.NewFile("cli")
	file.Func().Id("render").Params().Params(jen.Any(), jen.Any(), jen.Error()).Block(stmt)
	var b strings.Builder
	if err := file.Render(&b); err != nil {
		panic(err)
	}
	src := b.String()
	marker := "func render() {\n"
	idx := strings.Index(src, marker)
	if idx == -1 {
		return src
	}
	body := src[idx+len(marker):]
	body = strings.TrimSuffix(body, "}\n")
	return body
}
