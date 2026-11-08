package openapiimport

import (
	"encoding/json/v2"
	"errors"
	"strconv"
	"strings"
)

type sourceDisplayError struct {
	cause   error
	message string
}

// RedactSourceError removes token-bounded URI userinfo from public error prose.
// Known URI producers must project whole values before formatting, including
// rejected malformed values. It preserves errors.Is classification
// without exposing the original secret-bearing error through Unwrap. Internal
// resolution and diagnostic selection must finish before this projection.
func RedactSourceError(err error) error {
	if err == nil {
		return nil
	}
	message := sourceDisplayText(err.Error())
	if message == err.Error() {
		return err
	}
	return &sourceDisplayError{cause: err, message: message}
}

func (e *sourceDisplayError) Error() string {
	return e.message
}
func (e *sourceDisplayError) Is(target error) bool {
	return errors.Is(e.cause, target)
}

// sourceDisplayText preserves prose boundaries, including inside quotes. URI
// producers must project whole typed values before formatting them as prose.
func sourceDisplayText(text string) string {
	var result strings.Builder
	for start := 0; start < len(text); {
		if text[start] <= ' ' {
			result.WriteByte(text[start])
			start++
			continue
		}
		end := start + 1
		if text[start] == '"' || text[start] == '\'' {
			quote := text[start]
			for end < len(text) && text[end] != quote {
				if text[end] == '\\' && end+1 < len(text) {
					end++
				}
				end++
			}
			if end < len(text) {
				end++
			}
			result.WriteString(sourceDisplayQuoted(text[start:end]))
		} else {
			for end < len(text) && text[end] > ' ' && text[end] != '"' && text[end] != '\'' {
				end++
			}
			result.WriteString(sourceDisplayWord(text[start:end]))
		}
		start = end
	}
	return result.String()
}

func sourceDisplayQuoted(text string) string {
	if len(text) < 2 || text[len(text)-1] != text[0] {
		return sourceDisplayWord(text)
	}
	value := text[1 : len(text)-1]
	if text[0] == '"' {
		if err := json.Unmarshal([]byte(text), &value); err != nil {
			decoded, decodeErr := strconv.Unquote(text)
			if decodeErr == nil {
				value = decoded
			}
		}
	}
	projected := sourceDisplayText(value)
	if projected == value {
		return text
	}
	if text[0] == '"' {
		return strconv.Quote(projected)
	}
	return "'" + projected + "'"
}

func sourceDisplayWord(value string) string {
	if strings.HasPrefix(value, "#/") {
		return sourceDisplayPointer(value)
	}
	return sourceDisplayURI(value)
}

// sourceDisplayPointer decodes each pointer token independently. A raw slash
// separates source fields; it can never extend a URI authority in another token.
// Only explicit authored URI metadata grants whole-token URI interpretation.
// Other tokens are prose-bounded and retain their original bytes when unchanged.
func sourceDisplayPointer(pointer string, authored ...string) string {
	tokens := strings.Split(pointer, "/")
	for i, token := range tokens {
		value := strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
		projected := sourceDisplayText(value)
		for _, uri := range authored {
			if uri != "" && value == uri {
				projected = sourceDisplayURI(value)
			}
		}
		if projected != value {
			tokens[i] = escapeJSONPointer(projected)
		}
	}
	return strings.Join(tokens, "/")
}

// sourceDisplayURI removes authority userinfo without parsing. Malformed URI
// escapes therefore receive the same privacy as valid URIs. Callers own the
// surrounding URI, pointer-token or text-token boundary.
func sourceDisplayURI(value string) string {
	start := 0
	for start < len(value) {
		offset := strings.Index(value[start:], "//")
		if offset < 0 {
			break
		}
		marker := start + offset
		authority := marker + 2
		if marker > 0 && value[marker-1] != ':' && !strings.ContainsRune("=(<[", rune(value[marker-1])) {
			start = authority
			continue
		}
		end := authority
		for end < len(value) && !strings.ContainsRune("/?#", rune(value[end])) {
			end++
		}
		if at := strings.LastIndexByte(value[authority:end], '@'); at >= 0 {
			value = value[:authority] + "[redacted]" + value[authority+at:]
			start = authority + len("[redacted]@")
		} else {
			start = end
		}
	}
	return value
}

func publicSourceLocation(location SourceLocation) SourceLocation {
	location.Document = sourceDisplayText(location.Document)
	location.Pointer = sourceDisplayPointer(location.Pointer, location.Authored)
	location.Authored = sourceDisplayURI(location.Authored)
	return location
}

func publicDiagnostics(diagnostics Diagnostics, bundle *bundledDescription) Diagnostics {
	if diagnostics == nil {
		return nil
	}
	result := append(Diagnostics(nil), diagnostics...)
	if bundle != nil {
		bundle.attachOrigins(result)
	}
	for i, diagnostic := range result {
		diagnostic.Code = sourceDisplayText(diagnostic.Code)
		authored := ""
		if diagnostic.Origin != nil {
			authored = diagnostic.Origin.Authored
		}
		diagnostic.Path = sourceDisplayPointer(diagnostic.Path, authored)
		diagnostic.Message = sourceDisplayText(diagnostic.Message)
		if diagnostic.Origin != nil {
			origin := publicSourceLocation(*diagnostic.Origin)
			diagnostic.Origin = &origin
		}
		result[i] = diagnostic
	}
	return result
}

func (a *PartialAnalysis) projectPublicDiagnostics() {
	if a == nil {
		return
	}
	var bundle *bundledDescription
	if a.Document != nil {
		bundle = a.Document.bundle
	}
	a.Blocked = publicDiagnostics(a.Blocked, bundle)
	a.Warnings = publicDiagnostics(a.Warnings, bundle)
	a.Omitted = publicDiagnostics(a.Omitted, bundle)
	for _, operations := range [][]SkippedOperation{a.OperationOmissions, a.Skipped} {
		for i := range operations {
			operations[i].Method = sourceDisplayText(operations[i].Method)
			operations[i].Path = sourceDisplayText(operations[i].Path)
			operations[i].Diagnostics = publicDiagnostics(operations[i].Diagnostics, bundle)
		}
	}
}
