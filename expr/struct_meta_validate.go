package expr

import (
	"errors"
	"strings"
	"unicode/utf8"

	"golang.org/x/mod/module"

	"github.com/CaliLuke/loom/eval"
)

// validateStructMeta validates the struct:pkg:path and struct:name:proto
// metadata values of meta. Code generation escapes non-ASCII runes in package
// paths and treats them as word separators in protocol buffer message names,
// so these checks reject only values that stay invalid after that mapping.
func validateStructMeta(ctx string, meta MetaExpr, parent eval.Expression) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	for _, value := range meta["struct:pkg:path"] {
		if value == "" {
			continue
		}
		if reason := invalidStructPkgPath(value); reason != "" {
			verr.Add(parent, "%smetadata %q value %q is not a valid relative Go import path: %s", ctx, "struct:pkg:path", value, reason)
		}
	}
	for _, value := range meta["struct:name:proto"] {
		if !validProtoMessageName(value) {
			verr.Add(parent, "%smetadata %q value %q is not a valid protocol buffer message name: use ASCII letters, digits and underscores, starting with a letter or underscore", ctx, "struct:name:proto", value)
		}
	}
	return verr
}

// invalidStructPkgPath returns why path cannot locate a generated package, or
// the empty string when it can. Code generation escapes each non-ASCII rune of
// path into ASCII letters and digits, so the check replaces those runes with
// a letter before applying the Go import path rules.
func invalidStructPkgPath(path string) string {
	ascii := strings.Map(func(r rune) rune {
		if r >= utf8.RuneSelf {
			return 'u'
		}
		return r
	}, path)
	err := module.CheckImportPath(ascii)
	if err == nil {
		return ""
	}
	var pathErr *module.InvalidPathError
	if errors.As(err, &pathErr) && pathErr.Err != nil {
		return pathErr.Err.Error()
	}
	return err.Error()
}

// validProtoMessageName reports whether name can name a protocol buffer
// message. ASCII runes must be letters, digits or underscores. An ASCII name
// must also start with a letter or underscore; in any other name, code
// generation treats the non-ASCII runes as word separators and prefixes a
// digit-leading result.
func validProtoMessageName(name string) bool {
	if name == "" {
		return false
	}
	ascii := true
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= utf8.RuneSelf:
			ascii = false
		case c == '_', 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9':
		default:
			return false
		}
	}
	return !ascii || name[0] < '0' || name[0] > '9'
}
