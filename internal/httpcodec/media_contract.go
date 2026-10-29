package httpcodec

import (
	"fmt"
	"mime"
	"strings"
)

type mediaContract struct {
	base   string
	major  string
	params map[string]string
	rank   int
}

// ValidateMediaContracts rejects malformed, duplicate and equally specific
// overlapping contracts. Exact types outrank subtype ranges, which outrank */*.
func ValidateMediaContracts(contracts []string) error {
	parsed := make([]mediaContract, len(contracts))
	for i, value := range contracts {
		item, err := parseMediaContract(value, true)
		if err != nil {
			return err
		}
		for j := range i {
			other := parsed[j]
			if item.base == other.base && len(item.params) == len(other.params) && compatibleMediaParameters(item.params, other.params) {
				return fmt.Errorf("ambiguous media contracts %q and %q", contracts[j], value)
			}
		}
		parsed[i] = item
	}
	return nil
}

// SelectMediaContract selects the most specific matching contract for a concrete
// Content-Type. Parameters in the contract must match the received parameters;
// additional received parameters do not prevent a match. Missing, malformed,
// ambiguous and unmatched content types return an error.
func SelectMediaContract(contracts []string, contentType string) (int, error) {
	received, err := parseMediaContract(contentType, false)
	if err != nil {
		return -1, err
	}
	selected, bestRank, bestParams := -1, -1, -1
	for i, value := range contracts {
		candidate, err := parseMediaContract(value, true)
		if err != nil {
			return -1, err
		}
		if candidate.base != received.base && candidate.base != "*/*" && candidate.base != received.major+"/*" {
			continue
		}
		matches := true
		for key, value := range candidate.params {
			if receivedValue, exists := received.params[key]; !exists || receivedValue != value {
				matches = false
				break
			}
		}
		if !matches {
			continue
		}
		if candidate.rank < bestRank || candidate.rank == bestRank && len(candidate.params) < bestParams {
			continue
		}
		if candidate.rank == bestRank && len(candidate.params) == bestParams {
			return -1, fmt.Errorf("ambiguous media contract for %q", contentType)
		}
		selected, bestRank, bestParams = i, candidate.rank, len(candidate.params)
	}
	if selected < 0 {
		return -1, fmt.Errorf("no media contract for %q", contentType)
	}
	return selected, nil
}

func parseMediaContract(value string, allowRange bool) (mediaContract, error) {
	base, params, err := mime.ParseMediaType(value)
	major, minor, ok := strings.Cut(base, "/")
	if err != nil || !ok || major == "" || minor == "" || strings.Contains(minor, "/") {
		return mediaContract{}, fmt.Errorf("invalid media type %q", value)
	}
	if charset, exists := params["charset"]; exists {
		params["charset"] = strings.ToLower(charset)
	}
	rank := 3
	if strings.Contains(base, "*") {
		switch {
		case !allowRange:
			return mediaContract{}, fmt.Errorf("Content-Type must be concrete: %q", value)
		case base == "*/*":
			rank = 1
		case major != "*" && !strings.Contains(major, "*") && minor == "*":
			rank = 2
		default:
			return mediaContract{}, fmt.Errorf("invalid media range %q", value)
		}
	}
	return mediaContract{base: base, major: major, params: params, rank: rank}, nil
}

func compatibleMediaParameters(a, b map[string]string) bool {
	for key, value := range a {
		if other, exists := b[key]; exists && other != value {
			return false
		}
	}
	return true
}
