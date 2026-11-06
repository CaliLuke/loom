package http

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// AddRequestCookieParam validates the entire binding and its key ownership before
// adding verbatim cookie pairs. It leaves headers unchanged on rejection.
func AddRequestCookieParam(req *http.Request, name string, codec ParamCodec, value ParamValue, siblings ParamNamespace) error {
	if err := validateCookieCodec(name, codec, value.Kind); err != nil {
		return err
	}
	if req == nil {
		return fmt.Errorf("parameter %q: cookie request is nil", name)
	}
	if !value.Present {
		return nil
	}
	escape, err := cookieParamEscape(name, codec, value)
	if err != nil {
		return err
	}
	var cookies []string
	add := func(key, text string) error {
		if paramSiblingOwns(siblings, key) {
			return paramError(name, "encoded key belongs to another binding")
		}
		if err := validateCookiePair(key, text); err != nil {
			return paramError(name, "invalid cookie pair")
		}
		cookies = append(cookies, key+"="+text)
		return nil
	}
	switch {
	case value.Kind == ParamPairs && codec.Explode:
		seen := make(map[string]bool)
		for _, pair := range value.Pairs {
			if seen[pair.Key] {
				return paramError(name, "duplicate associative key")
			}
			seen[pair.Key] = true
			if err := add(pair.Key, pair.Value); err != nil {
				return err
			}
		}
	case value.Kind == ParamList && codec.Explode:
		for _, item := range value.Items {
			if err := add(name, item); err != nil {
				return err
			}
		}
	default:
		if value.Kind == ParamList && len(value.Items) == 1 && value.Items[0] == "" {
			return paramError(name, "singleton empty element is indistinguishable from an empty list")
		}
		parts, err := paramParts(name, value, false, escape)
		if err != nil {
			return err
		}
		if err := add(name, strings.Join(parts, ",")); err != nil {
			return err
		}
	}
	publishRequestCookies(req, cookies)
	return nil
}

// DecodeCookieParam decodes cookie pairs, retaining all occurrences for exploded
// sequences and the first occurrence for each scalar or associative name.
func DecodeCookieParam(cookies []*http.Cookie, name string, codec ParamCodec, kind ParamKind, siblings ParamNamespace) (ParamValue, error) {
	if err := validateCookieCodec(name, codec, kind); err != nil {
		return ParamValue{}, err
	}
	value := ParamValue{Kind: kind}
	if kind == ParamPairs && codec.Explode {
		seen := make(map[string]bool)
		for _, cookie := range cookies {
			if cookie == nil || !paramOwns(siblings, cookie.Name) || seen[cookie.Name] {
				continue
			}
			seen[cookie.Name] = true
			value.Present = true
			value.Pairs = append(value.Pairs, ParamPair{Key: cookie.Name, Value: cookie.Value})
		}
		return value, nil
	}
	var texts []string
	for _, cookie := range cookies {
		if cookie != nil && cookie.Name == name {
			texts = append(texts, cookie.Value)
			if kind != ParamList || !codec.Explode {
				break
			}
		}
	}
	if len(texts) == 0 {
		return value, nil
	}
	unescape := paramNoUnescape
	if codec.Style == ParamForm {
		unescape = url.PathUnescape
	}
	if kind == ParamScalar || codec.Explode {
		return decodeParamParts(name, texts, kind, false, unescape)
	}
	var parts []string
	if texts[0] != "" {
		parts = strings.Split(texts[0], ",")
	}
	return decodeParamParts(name, parts, kind, false, unescape)
}

func validateCookieCodec(name string, codec ParamCodec, kind ParamKind) error {
	if !paramKindValid(kind) || codec.Style != ParamCookie && codec.Style != ParamForm || codec.Style == ParamForm && codec.Explode && kind != ParamScalar {
		return fmt.Errorf("parameter %q: invalid cookie codec", name)
	}
	return nil
}

func cookieParamData(text string) bool {
	for i := 0; i < len(text); i++ {
		if text[i] < 0x20 || text[i] > 0x7e || strings.ContainsRune("\";,\\", rune(text[i])) {
			return false
		}
	}
	return true
}

func validateCookiePair(name, value string) error {
	cookie := &http.Cookie{Name: name, Value: value}
	if cookie.Valid() != nil || strings.Trim(value, " \t") != value {
		return paramError(name, "invalid cookie pair")
	}
	return nil
}

func publishRequestCookies(req *http.Request, pairs []string) {
	if len(pairs) == 0 {
		return
	}
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	all := append([]string(nil), req.Header.Values("Cookie")...)
	all = append(all, pairs...)
	req.Header.Set("Cookie", strings.Join(all, "; "))
}

func cookieParamEscape(name string, codec ParamCodec, value ParamValue) (func(string) string, error) {
	escape := paramIdentity
	if codec.Style == ParamForm {
		escape = func(text string) string {
			return strings.ReplaceAll(url.QueryEscape(text), "+", "%20")
		}
	} else {
		tokens, err := paramParts(name, value, false, paramIdentity)
		if err != nil {
			return nil, err
		}
		for _, token := range tokens {
			if !cookieParamData(token) {
				return nil, paramError(name, "value cannot be represented as cookie data")
			}
		}
	}
	return escape, nil
}
