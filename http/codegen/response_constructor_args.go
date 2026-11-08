package codegen

func responseConstructorArgRefs(args []*InitArgData, status string) []string {
	refs := make([]string, len(args))
	for i, arg := range args {
		refs[i] = arg.Ref
		if arg.responseStatus {
			refs[i] = status
		}
	}
	return refs
}
