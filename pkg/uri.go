package loom

import (
	"fmt"
	"net"
	"strings"
)

func validateRFC3986URI(value string, allowRelative bool) error {
	if isRFC3986URI(value, allowRelative) {
		return nil
	}
	kind := "URI"
	if allowRelative {
		kind = "URI-reference"
	}
	return fmt.Errorf("%q is not a valid RFC 3986 %s", value, kind)
}

func isRFC3986URI(value string, allowRelative bool) bool {
	remainder := value
	if fragment := strings.IndexByte(remainder, '#'); fragment >= 0 {
		if !validURIQueryOrFragment(remainder[fragment+1:]) {
			return false
		}
		remainder = remainder[:fragment]
	}
	if query := strings.IndexByte(remainder, '?'); query >= 0 {
		if !validURIQueryOrFragment(remainder[query+1:]) {
			return false
		}
		remainder = remainder[:query]
	}

	if colon := strings.IndexByte(remainder, ':'); colon > 0 {
		slash := strings.IndexByte(remainder, '/')
		if (slash < 0 || colon < slash) && validURIScheme(remainder[:colon]) {
			return validURIHierPart(remainder[colon+1:])
		}
	}
	return allowRelative && validURIRelativePart(remainder)
}

func validURIScheme(value string) bool {
	if len(value) == 0 || !isASCIIAlpha(value[0]) {
		return false
	}
	for i := 1; i < len(value); i++ {
		if !isASCIIAlpha(value[i]) && !isASCIIDigit(value[i]) && value[i] != '+' && value[i] != '-' && value[i] != '.' {
			return false
		}
	}
	return true
}

func validURIHierPart(value string) bool {
	if strings.HasPrefix(value, "//") {
		return validURIAuthorityAndPath(value[2:])
	}
	return validURIPath(value, false)
}

func validURIRelativePart(value string) bool {
	if strings.HasPrefix(value, "//") {
		return validURIAuthorityAndPath(value[2:])
	}
	if value == "" || value[0] == '/' {
		return validURIPath(value, false)
	}
	return validURIPath(value, true)
}

func validURIAuthorityAndPath(value string) bool {
	authority := value
	path := ""
	if slash := strings.IndexByte(value, '/'); slash >= 0 {
		authority = value[:slash]
		path = value[slash:]
	}
	return validURIAuthority(authority) && validURIPath(path, false)
}

func validURIAuthority(value string) bool {
	hostPort := value
	if at := strings.LastIndexByte(value, '@'); at >= 0 {
		if !validURIComponent(value[:at], isURIUserInfoChar) {
			return false
		}
		hostPort = value[at+1:]
	}

	if strings.HasPrefix(hostPort, "[") {
		closing := strings.IndexByte(hostPort, ']')
		if closing < 0 || !validURIIPLiteral(hostPort[1:closing]) {
			return false
		}
		return validURIPortSuffix(hostPort[closing+1:])
	}
	if strings.ContainsAny(hostPort, "[]") {
		return false
	}

	host := hostPort
	if colon := strings.LastIndexByte(hostPort, ':'); colon >= 0 {
		if strings.IndexByte(hostPort[:colon], ':') >= 0 || !validURIPort(hostPort[colon+1:]) {
			return false
		}
		host = hostPort[:colon]
	}
	return validURIComponent(host, isURIRegNameChar)
}

func validURIIPLiteral(value string) bool {
	if len(value) > 0 && (value[0] == 'v' || value[0] == 'V') {
		dot := strings.IndexByte(value, '.')
		if dot < 2 || dot == len(value)-1 {
			return false
		}
		for i := 1; i < dot; i++ {
			if !isASCIIHex(value[i]) {
				return false
			}
		}
		for i := dot + 1; i < len(value); i++ {
			if !isURIUnreserved(value[i]) && !isURISubDelimiter(value[i]) && value[i] != ':' {
				return false
			}
		}
		return true
	}
	return strings.Contains(value, ":") && net.ParseIP(value) != nil
}

func validURIPortSuffix(value string) bool {
	if value == "" {
		return true
	}
	return value[0] == ':' && validURIPort(value[1:])
}

func validURIPort(value string) bool {
	for i := 0; i < len(value); i++ {
		if !isASCIIDigit(value[i]) {
			return false
		}
	}
	return true
}

func validURIPath(value string, firstSegmentNoColon bool) bool {
	firstSegment := true
	for i := 0; i < len(value); i++ {
		switch value[i] {
		case '/':
			firstSegment = false
		case '%':
			if i+2 >= len(value) || !isASCIIHex(value[i+1]) || !isASCIIHex(value[i+2]) {
				return false
			}
			i += 2
		default:
			if !isURIPChar(value[i]) || firstSegmentNoColon && firstSegment && value[i] == ':' {
				return false
			}
		}
	}
	return !firstSegmentNoColon || len(value) > 0
}

func validURIQueryOrFragment(value string) bool {
	return validURIComponent(value, func(char byte) bool {
		return isURIPChar(char) || char == '/' || char == '?'
	})
}

func validURIComponent(value string, allowed func(byte) bool) bool {
	for i := 0; i < len(value); i++ {
		if value[i] == '%' {
			if i+2 >= len(value) || !isASCIIHex(value[i+1]) || !isASCIIHex(value[i+2]) {
				return false
			}
			i += 2
			continue
		}
		if !allowed(value[i]) {
			return false
		}
	}
	return true
}

func isURIUserInfoChar(char byte) bool {
	return isURIUnreserved(char) || isURISubDelimiter(char) || char == ':'
}

func isURIRegNameChar(char byte) bool {
	return isURIUnreserved(char) || isURISubDelimiter(char)
}

func isURIPChar(char byte) bool {
	return isURIUnreserved(char) || isURISubDelimiter(char) || char == ':' || char == '@'
}

func isURIUnreserved(char byte) bool {
	return isASCIIAlpha(char) || isASCIIDigit(char) || char == '-' || char == '.' || char == '_' || char == '~'
}

func isURISubDelimiter(char byte) bool {
	return strings.ContainsRune("!$&'()*+,;=", rune(char))
}

func isASCIIAlpha(char byte) bool {
	return char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z'
}

func isASCIIDigit(char byte) bool {
	return char >= '0' && char <= '9'
}

func isASCIIHex(char byte) bool {
	return isASCIIDigit(char) || char >= 'a' && char <= 'f' || char >= 'A' && char <= 'F'
}
