package loom

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"regexp/syntax"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidateFormat(t *testing.T) {
	var (
		validDate              = "2015-10-26"
		invalidDate            = "201510-26"
		validDateTime          = "2015-10-26T08:31:23Z"
		invalidDateTime        = "201510-26T08:31:23Z"
		validUUID              = "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
		validUUIDWithBrace     = "{6ba7b810-9dad-11d1-80b4-00c04fd430c8}"
		validUUIDRaw           = "6ba7b8109dad11d180b400c04fd430c8"
		validUUIDWithURNPrefix = "urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8"
		invalidUUID            = "96054a62-a9e45ed26688389b"
		invalidUUIDNonHex      = "abcdefgh-ijkl-mnop-qrst-uvqxyz012345" // UUID with characters other than hex digit
		validEmail             = "raphael@loom.design"

		// Re-enable once CircleCI uses Go 1.13
		// invalidEmail    = "foo"

		validHostname             = "loom.design"
		invalidHostname           = "_hi_"
		validIPv4                 = "192.168.0.1"
		invalidIPv4               = "192-168.0.1"
		validIPv6                 = "::1"
		invalidIPv6               = "foo"
		validURI                  = "hhp://loom.design/contact"
		invalidURI                = "foo_"
		validRelativeURIReference = "/download/token"
		validFragmentURIReference = "#result"
		invalidURIReference       = "%zz"
		validMAC                  = "06-00-00-00-00-00"
		invalidMAC                = "bar"
		validCIDR                 = "10.0.0.0/8"
		invalidCIDR               = "foo"
		validRegexp               = "^loom$"
		invalidRegexp             = "foo["
		validJSON                 = `{"a":"b","c":2}`
		invalidJSON               = "{"
		validRFC1123              = "Mon, 04 Jun 2017 23:52:05 MST"
		invalidRFC1123            = "Mon 04 Jun 2017 23:52:05 MST"
	)
	cases := map[string]struct {
		name     string
		val      string
		format   Format
		expected error
	}{
		"valid date":                 {"validDate", validDate, FormatDate, nil},
		"invalid date":               {"invalidDate", invalidDate, FormatDate, InvalidFormatError("invalidDate", invalidDate, FormatDate, &time.ParseError{Layout: time.DateOnly, Value: invalidDate, LayoutElem: "-", ValueElem: invalidDate[4:]})},
		"valid date-time":            {"validDateTime", validDateTime, FormatDateTime, nil},
		"invalid date-time":          {"invalidDateTime", invalidDateTime, FormatDateTime, InvalidFormatError("invalidDateTime", invalidDateTime, FormatDateTime, &time.ParseError{Layout: time.RFC3339, Value: invalidDateTime, LayoutElem: "-", ValueElem: invalidDateTime[4:]})},
		"valid uuid":                 {"validUUID", validUUID, FormatUUID, nil},
		"valid uuid with brace":      {"validUUIDWithBrace", validUUIDWithBrace, FormatUUID, nil},
		"valid uuid with no dash":    {"validUUIDRaw", validUUIDRaw, FormatUUID, nil},
		"valid uuid with urn prefix": {"validUUIDWithURNPrefix", validUUIDWithURNPrefix, FormatUUID, nil},
		"invalid uuid":               {"invalidUUID", invalidUUID, FormatUUID, InvalidFormatError("invalidUUID", invalidUUID, FormatUUID, fmt.Errorf("uuid: %s: invalid uuid", invalidUUID))},
		"invalid uuid non hex":       {"invalidUUIDNonHex", invalidUUIDNonHex, FormatUUID, InvalidFormatError("invalidUUIDNonHex", invalidUUIDNonHex, FormatUUID, fmt.Errorf("uuid: %s: invalid uuid", invalidUUIDNonHex))},

		"valid email": {"validEmail", validEmail, FormatEmail, nil},

		// Re-enable once CircleCI uses Go 1.13
		// "invalid email":      {"invalidEmail", invalidEmail, FormatEmail, InvalidFormatError("invalidEmail", invalidEmail, FormatEmail, errors.New("mail: missing '@' or angle-addr"))},

		"valid hostname":               {"validHostname", validHostname, FormatHostname, nil},
		"invalid hostname":             {"invalidHostname", invalidHostname, FormatHostname, InvalidFormatError("invalidHostname", invalidHostname, FormatHostname, fmt.Errorf("hostname value '%s' does not match %s", invalidHostname, `^[[:alnum:]][[:alnum:]\-]{0,61}[[:alnum:]]|[[:alpha:]]$`))},
		"valid ipv4":                   {"validIPv4", validIPv4, FormatIPv4, nil},
		"valid ipv6 as ipv4":           {"validIPv6", validIPv6, FormatIPv4, InvalidFormatError("validIPv6", validIPv6, FormatIPv4, fmt.Errorf("\"%s\" is an invalid %s value", validIPv6, FormatIPv4))},
		"invalid ipv4":                 {"invalidIPv4", invalidIPv4, FormatIPv4, InvalidFormatError("invalidIPv4", invalidIPv4, FormatIPv4, fmt.Errorf("\"%s\" is an invalid %s value", invalidIPv4, FormatIPv4))},
		"valid ipv6":                   {"validIPv6", validIPv6, FormatIPv6, nil},
		"valid ipv4 as ipv6":           {"validIPv4", validIPv4, FormatIPv6, InvalidFormatError("validIPv4", validIPv4, FormatIPv6, fmt.Errorf("\"%s\" is an invalid %s value", validIPv4, FormatIPv6))},
		"invalid ipv6":                 {"invalidIPv6", invalidIPv6, FormatIPv6, InvalidFormatError("invalidIPv6", invalidIPv6, FormatIPv6, fmt.Errorf("\"%s\" is an invalid %s value", invalidIPv6, FormatIPv6))},
		"valid ipv4 as ip":             {"validIPv4", validIPv4, FormatIP, nil},
		"valid ipv6 as ip":             {"validIPv6", validIPv6, FormatIP, nil},
		"invalid ipv4 as ip":           {"invalidIPv4", invalidIPv4, FormatIP, InvalidFormatError("invalidIPv4", invalidIPv4, FormatIP, fmt.Errorf("\"%s\" is an invalid %s value", invalidIPv4, FormatIP))},
		"invalid ipv6 as ip":           {"invalidIPv6", invalidIPv6, FormatIP, InvalidFormatError("invalidIPv6", invalidIPv6, FormatIP, fmt.Errorf("\"%s\" is an invalid %s value", invalidIPv6, FormatIP))},
		"valid uri":                    {"validURI", validURI, FormatURI, nil},
		"invalid uri":                  {"invalidURI", invalidURI, FormatURI, InvalidFormatError("invalidURI", invalidURI, FormatURI, fmt.Errorf("%q is not a valid RFC 3986 URI", invalidURI))},
		"valid relative uri-reference": {"validRelativeURIReference", validRelativeURIReference, FormatURIReference, nil},
		"valid fragment uri-reference": {"validFragmentURIReference", validFragmentURIReference, FormatURIReference, nil},
		"invalid uri-reference":        {"invalidURIReference", invalidURIReference, FormatURIReference, InvalidFormatError("invalidURIReference", invalidURIReference, FormatURIReference, fmt.Errorf("%q is not a valid RFC 3986 URI-reference", invalidURIReference))},
		"valid mac":                    {"validMAC", validMAC, FormatMAC, nil},
		"invalid mac":                  {"invalidMAC", invalidMAC, FormatMAC, InvalidFormatError("invalidMAC", invalidMAC, FormatMAC, &net.AddrError{Err: "invalid MAC address", Addr: invalidMAC})},
		"valid cidr":                   {"validCIDR", validCIDR, FormatCIDR, nil},
		"invalid cidr":                 {"invalidCIDR", invalidCIDR, FormatCIDR, InvalidFormatError("invalidCIDR", invalidCIDR, FormatCIDR, &net.ParseError{Type: "CIDR address", Text: invalidCIDR})},
		"valid regexp":                 {"validRegexp", validRegexp, FormatRegexp, nil},
		"invalid regexp":               {"invalidRegexp", invalidRegexp, FormatRegexp, InvalidFormatError("invalidRegexp", invalidRegexp, FormatRegexp, &syntax.Error{Code: syntax.ErrMissingBracket, Expr: invalidRegexp[3:4]})},
		"valid json":                   {"validJSON", validJSON, FormatJSON, nil},
		"invalid json":                 {"invalidJSON", invalidJSON, FormatJSON, InvalidFormatError("invalidJSON", invalidJSON, FormatJSON, fmt.Errorf("invalid JSON"))},
		"valid rfc1123":                {"validRFC1123", validRFC1123, FormatRFC1123, nil},
		"invalid rfc1123":              {"invalidRFC1123", invalidRFC1123, FormatRFC1123, InvalidFormatError("invalidRFC1123", invalidRFC1123, FormatRFC1123, &time.ParseError{Layout: time.RFC1123, Value: invalidRFC1123, LayoutElem: ", ", ValueElem: invalidRFC1123[3:]})},
	}

	for k, tc := range cases {
		actual := ValidateFormat(tc.name, tc.val, tc.format)
		if !errors.Is(actual, tc.expected) {
			// Compare only the messages because the error has always a new error ID.
			if actual == nil || tc.expected == nil || actual.Error() != tc.expected.Error() {
				t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
			}
		}
	}
}

func TestValidateURIFormatsRFC3986(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		format Format
		valid  bool
	}{
		{name: "URI with authority", value: "https://example.com/download?q=one#part", format: FormatURI, valid: true},
		{name: "opaque URI", value: "urn:example:animal:ferret:nose", format: FormatURI, valid: true},
		{name: "URI with IPvFuture", value: "https://[v1.fe80::a]/", format: FormatURI, valid: true},
		{name: "absolute URI-reference", value: "https://example.com/download", format: FormatURIReference, valid: true},
		{name: "network URI-reference", value: "//example.com/download", format: FormatURIReference, valid: true},
		{name: "absolute-path URI-reference", value: "/download/token", format: FormatURIReference, valid: true},
		{name: "relative-path URI-reference", value: "download/token", format: FormatURIReference, valid: true},
		{name: "query URI-reference", value: "?token=%2F", format: FormatURIReference, valid: true},
		{name: "fragment URI-reference", value: "#result", format: FormatURIReference, valid: true},
		{name: "empty URI-reference", value: "", format: FormatURIReference, valid: true},
		{name: "percent-encoded URI-reference", value: "/download/%2Ftoken", format: FormatURIReference, valid: true},
		{name: "IPvFuture URI-reference", value: "//[vF.Foo:bar]/resource", format: FormatURIReference, valid: true},
		{name: "URI without scheme", value: "/download/token", format: FormatURI},
		{name: "URI with invalid scheme", value: "1http://example.com", format: FormatURI},
		{name: "empty URI", value: "", format: FormatURI},
		{name: "raw space", value: "/download/a b", format: FormatURIReference},
		{name: "raw Unicode", value: "/download/café", format: FormatURIReference},
		{name: "control character", value: "/download/\n", format: FormatURIReference},
		{name: "malformed percent encoding", value: "/download/%zz", format: FormatURIReference},
		{name: "misplaced path delimiter", value: "/download/[token]", format: FormatURIReference},
		{name: "duplicate fragment delimiter", value: "/download#one#two", format: FormatURIReference},
		{name: "malformed IPv6 authority", value: "//[::1/resource", format: FormatURIReference},
		{name: "malformed IPvFuture authority", value: "//[v1.]/resource", format: FormatURIReference},
		{name: "nonnumeric authority port", value: "//example.com:bad/resource", format: FormatURIReference},
		{name: "duplicate userinfo delimiter", value: "https://user@@example.com/resource", format: FormatURI},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateFormat("location", test.value, test.format)
			if test.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestValidatePattern(t *testing.T) {
	var (
		name      = "foo"
		pattern   = "^loom$"
		matched   = "loom"
		unmatched = "foo["
	)
	cases := map[string]struct {
		name     string
		val      string
		pattern  string
		expected error
	}{
		"matched value":   {name, matched, pattern, nil},
		"unmatched value": {name, unmatched, pattern, InvalidPatternError(name, unmatched, pattern)},
	}

	for k, tc := range cases {
		actual := ValidatePattern(tc.name, tc.val, tc.pattern)
		if !errors.Is(actual, tc.expected) {
			// Compare only the messages because the error has always a new error ID.
			if actual == nil || tc.expected == nil || actual.Error() != tc.expected.Error() {
				t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
			}
		}
	}
}

func TestValidatePatternCompiled(t *testing.T) {
	var (
		name      = "foo"
		pattern   = regexp.MustCompile("^loom$")
		matched   = "loom"
		unmatched = "foo["
	)

	if actual := ValidatePatternCompiled(name, matched, pattern); actual != nil {
		t.Fatalf("matched value: got %#v, expected nil", actual)
	}

	actual := ValidatePatternCompiled(name, unmatched, pattern)
	expected := InvalidPatternError(name, unmatched, pattern.String())
	if !errors.Is(actual, expected) {
		if actual == nil || actual.Error() != expected.Error() {
			t.Fatalf("unmatched value: got %#v, expected %#v", actual, expected)
		}
	}
}

func TestJSONValueHelpers(t *testing.T) {
	const precise = "9007199254740993"
	value := JSONValueFromString(precise)
	if got := string(value); got != `"9007199254740993"` {
		t.Fatalf("JSONValueFromString() = %s", got)
	}
	if got := JSONValueString(value); got != precise {
		t.Fatalf("JSONValueString() = %q", got)
	}
	encoded, err := JSONValueFrom(int64(9007199254740993))
	if err != nil {
		t.Fatalf("JSONValueFrom() error = %v", err)
	}
	if got := string(encoded); got != precise {
		t.Fatalf("JSONValueFrom() = %s", got)
	}
	if _, err := JSONValueFrom(make(chan struct{})); err == nil {
		t.Fatal("JSONValueFrom accepted an unsupported value")
	}

	if !JSONValueEqual(JSONValue(`true`), true) {
		t.Fatal("JSONValueEqual rejected an equal boolean")
	}
	if !JSONValueEqual(JSONValue("1.0"), 1) {
		t.Fatal("JSONValueEqual rejected equivalent number spellings")
	}
	if JSONValueEqual(JSONValue("9007199254740993"), int64(9007199254740992)) {
		t.Fatal("JSONValueEqual rounded distinct large integers")
	}
	if JSONValueEqual(JSONValue("0.123456789012345678901"), 0.12345678901234568) {
		t.Fatal("JSONValueEqual rounded a precise decimal")
	}
	if !JSONValueEqual(JSONValue("12345678901234567890.0"), JSONValue("12345678901234567890")) {
		t.Fatal("JSONValueEqual rejected exact equivalent decimals")
	}
	if !JSONValueEqual(JSONValue("{ \"right\": 2, \"left\": 1 }"), map[string]int{"left": 1, "right": 2}) {
		t.Fatal("JSONValueEqual rejected equivalent object formatting")
	}
	if JSONValueEqual(JSONValue(`"true"`), true) {
		t.Fatal("JSONValueEqual accepted a JSON string as a boolean")
	}
}
