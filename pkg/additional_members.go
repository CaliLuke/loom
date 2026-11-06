package loom

import "fmt"

// CheckAdditionalMembers returns an error when an undeclared member of an
// object uses a name that the object declares. Encoding such a member beside
// the declared members would emit a duplicate name or replace a declared
// member that is omitted. path locates the object in error messages.
func CheckAdditionalMembers[V any](members map[string]V, path string, declared ...string) error {
	for _, name := range declared {
		if _, collides := members[name]; collides {
			return fmt.Errorf("%s: additional member %q collides with a declared member", path, name)
		}
	}
	return nil
}

// InvalidAdditionalMemberError is the validation error produced by the
// generated code when a decoded object carries, as an undeclared member, a
// member that the design declares but the transport binds elsewhere, such as
// to a path parameter or header. name is the path of the member. The error
// uses the InvalidFieldType name, as other errors about a value that does not
// fit the shape of its field do.
func InvalidAdditionalMemberError(name string) error {
	message := fmt.Sprintf("invalid member %q; a declared member cannot appear as an additional member", name)
	return validationErrorWithSafe(
		withField(name, PermanentError(InvalidFieldType, "%s", message)),
		message,
	)
}
