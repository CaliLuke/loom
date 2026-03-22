// Package cli contains helpers used by transport-specific command-line client
// generators for parsing the command-line flags to identify the service and
// the method to make a request along with the request payload to be sent.
package cli

import (
	"bytes"
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
		Conversion string
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
		Init string
	}

	// PayloadInitData contains the data needed to generate a constructor
	// function that initializes a service method payload type from the
	// command-ling arguments.
	PayloadInitData struct {
		// Code is the payload initialization code.
		Code string
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

	var conversion string
	if m.Payload != "" && buildFunction == nil && len(flags) > 0 {
		// No build function, just convert the arg to the body type
		var convPre, convSuff string
		target := "data"
		if flagType(m.Payload) == "JSON" {
			target = "val"
			convPre = fmt.Sprintf("var val %s\n", m.Payload)
			convSuff = "\ndata = val"
		}
		conv, _, check := conversionCode(
			"*"+flags[0].FullName+"Flag",
			target,
			m.Payload,
			false,
		)
		conversion = convPre + conv + convSuff
		if check {
			conversion = "var err error\n" + conversion
			conversion += "\nif err != nil {\n"
			if flagType(m.Payload) == "JSON" {
				conversion += fmt.Sprintf(`return nil, nil, fmt.Errorf("invalid JSON for %s, \nerror: %%s, \nexample of valid JSON:\n%%s", err, %q)`,
					flags[0].FullName+"Flag", flags[0].Example)
			} else {
				conversion += fmt.Sprintf(`return nil, nil, fmt.Errorf("invalid value for %s, must be %s")`,
					flags[0].FullName+"Flag", flags[0].Type)
			}
			conversion += "\n}"
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
	var flagsCode bytes.Buffer
	flagsCode.WriteString("var (\n")
	for _, cmd := range data {
		fmt.Fprintf(&flagsCode, "\t%sFlags = flag.NewFlagSet(%q, flag.ContinueOnError)\n", cmd.VarName, cmd.Name)
		flagsCode.WriteString("\n")
		for _, sub := range cmd.Subcommands {
			fmt.Fprintf(&flagsCode, "\t%sFlags = flag.NewFlagSet(%q, flag.ExitOnError)\n", sub.FullName, sub.Name)
			for _, flag := range sub.Flags {
				defaultValue := ""
				if flag.Default != nil {
					defaultValue = fmt.Sprint(flag.Default)
				} else if flag.Required {
					defaultValue = "REQUIRED"
				}
				fmt.Fprintf(&flagsCode, "\t%sFlag = %sFlags.String(%q, %q, %q)\n", flag.FullName, sub.FullName, flag.Name, defaultValue, flag.Description)
			}
			flagsCode.WriteString("\n")
		}
	}
	flagsCode.WriteString(")\n")

	for _, cmd := range data {
		fmt.Fprintf(&flagsCode, "%sFlags.Usage = %sUsage\n", cmd.VarName, cmd.VarName)
		for _, sub := range cmd.Subcommands {
			fmt.Fprintf(&flagsCode, "%sFlags.Usage = %sUsage\n", sub.FullName, sub.FullName)
		}
		flagsCode.WriteString("\n")
	}

	flagsCode.WriteString(`if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
	return nil, nil, err
}

if flag.NArg() < 2 { // two non flag args are required: SERVICE and ENDPOINT (aka COMMAND)
	return nil, nil, fmt.Errorf("not enough arguments")
}

var (
	svcn string
	svcf *flag.FlagSet
)
{
	svcn = flag.Arg(0)
	switch svcn {
`)
	for _, cmd := range data {
		fmt.Fprintf(&flagsCode, "\tcase %q:\n\t\tsvcf = %sFlags\n", cmd.Name, cmd.VarName)
	}
	flagsCode.WriteString(`	default:
		return nil, nil, fmt.Errorf("unknown service %q", svcn)
	}
}
if err := svcf.Parse(flag.Args()[1:]); err != nil {
	return nil, nil, err
}

var (
	epn string
	epf *flag.FlagSet
)
{
	epn = svcf.Arg(0)
	switch svcn {
`)
	for _, cmd := range data {
		fmt.Fprintf(&flagsCode, "\tcase %q:\n\t\tswitch epn {\n", cmd.Name)
		for _, sub := range cmd.Subcommands {
			fmt.Fprintf(&flagsCode, "\t\tcase %q:\n\t\t\tepf = %sFlags\n", sub.Name, sub.FullName)
			flagsCode.WriteString("\n")
		}
		flagsCode.WriteString("\t\t}\n\n")
	}
	flagsCode.WriteString(`	}
}
if epf == nil {
	return nil, nil, fmt.Errorf("unknown %q endpoint %q", svcn, epn)
}

// Parse endpoint flags if any
if svcf.NArg() > 1 {
	if err := epf.Parse(svcf.Args()[1:]); err != nil {
		return nil, nil, err
	}
}
`)

	return flagsCode.String()
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
				group.Block(codegen.Expr(field.Init))
			}
			if buildFunction.PayloadInit != nil {
				if buildFunction.PayloadInit.Code != "" {
					group.Add(codegen.Expr(buildFunction.PayloadInit.Code))
					if buildFunction.PayloadInit.ReturnTypeAttribute != "" {
						value := buildFunction.PayloadInit.ReturnTypeAttribute + ": "
						if buildFunction.PayloadInit.ReturnTypeAttributePointer {
							value += "&"
						}
						value += "v,\n"
						group.Add(codegen.Expr("res := &" + buildFunction.PayloadInit.ReturnTypeName + "{\n" + value + "}"))
					}
				}
				if buildFunction.PayloadInit.ReturnIsStruct {
					if buildFunction.PayloadInit.Code == "" {
						target := "v"
						if buildFunction.PayloadInit.ReturnTypeAttribute != "" {
							target = "res"
						}
						group.Add(codegen.Expr(target + " := &" + buildFunction.PayloadInit.ReturnTypeName + "{}"))
					}
					group.Add(codegen.Expr(fieldCode(buildFunction.PayloadInit)))
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
func FieldLoadCode(f *FlagData, argName, argTypeName, validate string, defaultValue any, payload expr.DataType, payloadRef string) (string, bool) {
	var (
		code    string
		declErr bool
		startIf string
		endIf   string
	)
	if !f.Required {
		startIf = fmt.Sprintf("if %s != \"\" {\n", f.FullName)
		endIf = "\n}"
	}
	if argTypeName == codegen.GoNativeTypeName(expr.String) {
		ref := "&"
		if f.Required || defaultValue != nil {
			ref = ""
		}
		code = argName + " = " + ref + f.FullName
		declErr = validate != ""
	} else {
		var checkErr bool
		code, declErr, checkErr = conversionCode(f.FullName, argName, argTypeName, !f.Required && defaultValue == nil)
		if checkErr {
			code += "\nif err != nil {\n"
			nilVal := "nil"
			if expr.IsPrimitive(payload) {
				code += fmt.Sprintf("var zero %s\n", payloadRef)
				nilVal = "zero"
			}
			if flagType(argTypeName) == "JSON" {
				code += fmt.Sprintf(`return %s, fmt.Errorf("invalid JSON for %s, \nerror: %%s, \nexample of valid JSON:\n%%s", err, %q)`,
					nilVal, argName, f.Example)
			} else {
				code += fmt.Sprintf(`return %s, fmt.Errorf("invalid value for %s, must be %s")`,
					nilVal, argName, f.Type)
			}
			code += "\n}"
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
		code += "\n" + validate + "\n"
		nilVal := "nil"
		if expr.IsPrimitive(payload) {
			code += fmt.Sprintf("var zero %s\n", payloadRef)
			nilVal = "zero"
		}
		code += fmt.Sprintf("if err != nil {\n\treturn %s, err\n}", nilVal)
	}
	return fmt.Sprintf("%s%s%s", startIf, code, endIf), declErr
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
func conversionCode(from, to, typeName string, pointer bool) (string, bool, bool) {
	var (
		parse string
		cast  string

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
		if pointer {
			parse = fmt.Sprintf("var %s bool\n", target)
		}
		parse += fmt.Sprintf("%s, err = strconv.ParseBool(%s)", target, from)
	case intN:
		parse = fmt.Sprintf("var v int64\nv, err = strconv.ParseInt(%s, 10, strconv.IntSize)", from)
		cast = fmt.Sprintf("%s %s= int(v)", target, decl)
	case int32N:
		parse = fmt.Sprintf("var v int64\nv, err = strconv.ParseInt(%s, 10, 32)", from)
		cast = fmt.Sprintf("%s %s= int32(v)", target, decl)
	case int64N:
		parse = fmt.Sprintf("%s, err %s= strconv.ParseInt(%s, 10, 64)", target, decl, from)
		declErr = decl == ""
	case uintN:
		parse = fmt.Sprintf("var v uint64\nv, err = strconv.ParseUint(%s, 10, strconv.IntSize)", from)
		cast = fmt.Sprintf("%s %s= uint(v)", target, decl)
	case uint32N:
		parse = fmt.Sprintf("var v uint64\nv, err = strconv.ParseUint(%s, 10, 32)", from)
		cast = fmt.Sprintf("%s %s= uint32(v)", target, decl)
	case uint64N:
		parse = fmt.Sprintf("%s, err %s= strconv.ParseUint(%s, 10, 64)", target, decl, from)
		declErr = decl == ""
	case float32N:
		parse = fmt.Sprintf("var v float64\nv, err = strconv.ParseFloat(%s, 32)", from)
		cast = fmt.Sprintf("%s %s= float32(v)", target, decl)
	case float64N:
		parse = fmt.Sprintf("%s, err %s= strconv.ParseFloat(%s, 64)", target, decl, from)
		declErr = decl == ""
	case stringN:
		parse = fmt.Sprintf("%s %s= %s", target, decl, from)
		declErr = false
		checkErr = false
	case bytesN:
		parse = fmt.Sprintf("%s %s= []byte(%s)", target, decl, from)
		declErr = false
		checkErr = false
	default:
		parse = fmt.Sprintf("err = json.Unmarshal([]byte(%s), &%s)", from, target)
	}
	if !needCast {
		return parse, declErr, checkErr
	}
	if cast != "" {
		parse = parse + "\n" + cast
	}
	if to != target {
		ref := ""
		if pointer {
			ref = "&"
		}
		parse += fmt.Sprintf("\n%s = %s%s", to, ref, target)
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
func fieldCode(init *PayloadInitData) string {
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
	return c
}
