package cli

import "github.com/CaliLuke/loom/codegen"

// AllocateCommandIdentifiers returns detached command data whose generated
// client import, usage function, flag-set, and flag-value identifiers are
// unique among reserved and the other command identifiers in the file.
func AllocateCommandIdentifiers(commands []*CommandData, reserved []string) []*CommandData {
	scope := codegen.NewNameScope()
	for _, name := range reserved {
		if name != "" {
			scope.Unique(name)
		}
	}

	allocated := cloneCommands(commands)
	for _, command := range allocated {
		command.PkgName = scope.Unique(command.PkgName)
		command.UsageName = scope.Unique(commandUsageName(command))
		command.FlagSetName = scope.Unique(commandFlagSetName(command))
		for _, subcommand := range command.Subcommands {
			subcommand.UsageName = scope.Unique(subcommandUsageName(subcommand))
			subcommand.FlagSetName = scope.Unique(subcommandFlagSetName(subcommand))
			for _, flag := range subcommand.Flags {
				flag.ValueName = scope.Unique(flagValueName(flag))
			}
			if subcommand.Conversion != nil {
				subcommand.Conversion = buildSubcommandConversion(subcommand.PayloadType, subcommand.BuildFunction, subcommand.Flags)
			}
		}
	}
	return allocated
}

func cloneCommands(commands []*CommandData) []*CommandData {
	cloned := make([]*CommandData, len(commands))
	for i, command := range commands {
		copyCommand := *command
		copyCommand.Subcommands = make([]*SubcommandData, len(command.Subcommands))
		if command.Interceptors != nil {
			interceptors := *command.Interceptors
			copyCommand.Interceptors = &interceptors
		}
		for j, subcommand := range command.Subcommands {
			copySubcommand := *subcommand
			copySubcommand.Flags = cloneFlags(subcommand.Flags)
			if subcommand.Interceptors != nil {
				interceptors := *subcommand.Interceptors
				copySubcommand.Interceptors = &interceptors
			}
			if subcommand.BuildFunction != nil {
				copySubcommand.BuildFunction = cloneBuildFunction(subcommand.BuildFunction)
			}
			copyCommand.Subcommands[j] = &copySubcommand
		}
		cloned[i] = &copyCommand
	}
	return cloned
}

func cloneFlags(flags []*FlagData) []*FlagData {
	cloned := make([]*FlagData, len(flags))
	for i, flag := range flags {
		copyFlag := *flag
		cloned[i] = &copyFlag
	}
	return cloned
}

func cloneBuildFunction(function *BuildFunctionData) *BuildFunctionData {
	cloned := *function
	cloned.ActualParams = append([]string(nil), function.ActualParams...)
	cloned.FormalParams = append([]string(nil), function.FormalParams...)
	cloned.Fields = append([]*FieldData(nil), function.Fields...)
	if function.PayloadInit != nil {
		payloadInit := *function.PayloadInit
		payloadInit.Args = append([]*codegen.InitArgData(nil), function.PayloadInit.Args...)
		cloned.PayloadInit = &payloadInit
	}
	return &cloned
}

func commandUsageName(command *CommandData) string {
	if command.UsageName != "" {
		return command.UsageName
	}
	return command.VarName + "Usage"
}

func commandFlagSetName(command *CommandData) string {
	if command.FlagSetName != "" {
		return command.FlagSetName
	}
	return command.VarName + "Flags"
}

func subcommandUsageName(subcommand *SubcommandData) string {
	if subcommand.UsageName != "" {
		return subcommand.UsageName
	}
	return subcommand.FullName + "Usage"
}

func subcommandFlagSetName(subcommand *SubcommandData) string {
	if subcommand.FlagSetName != "" {
		return subcommand.FlagSetName
	}
	return subcommand.FullName + "Flags"
}

func flagValueName(flag *FlagData) string {
	if flag.ValueName != "" {
		return flag.ValueName
	}
	return flag.FullName + "Flag"
}
