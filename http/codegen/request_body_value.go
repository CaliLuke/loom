package codegen

import "strings"

func requestClientBodyValue(request *RequestData) string {
	if request.ClientBody == nil {
		return "nil"
	}
	if init := request.ClientBody.Init; init != nil {
		args := make([]string, len(init.ClientArgs))
		for i, arg := range init.ClientArgs {
			args[i] = arg.VarName
			if arg.FieldPointer {
				args[i] = "&" + args[i]
			}
		}
		return init.Name + "(" + strings.Join(args, ", ") + ")"
	}
	value := "p"
	if request.PayloadAttr != "" {
		value += "." + request.PayloadAttr
	}
	return value
}
