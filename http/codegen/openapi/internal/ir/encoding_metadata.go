package ir

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// encodingMetadataNode records the encoding families at one media or encoding
// object. Child objects have independent choices; positional prefix and item
// encodings can coexist, but neither can coexist with named encoding.
type encodingMetadataNode struct {
	named      bool
	positional bool
	children   map[string]*encodingMetadataNode
}

func validateOperationEncodings(request *RequestBody, responses map[string]*Response) error {
	if request != nil {
		if err := validateContentEncodings(request.Content); err != nil {
			return fmt.Errorf("request body: %w", err)
		}
	}
	for _, status := range slices.Sorted(maps.Keys(responses)) {
		if err := validateContentEncodings(responses[status].Content); err != nil {
			return fmt.Errorf("response %s: %w", status, err)
		}
	}
	return nil
}

func validateContentEncodings(content map[string]*MediaType) error {
	for _, name := range slices.Sorted(maps.Keys(content)) {
		if err := validateEncodingMetadata(content[name].Metadata); err != nil {
			return fmt.Errorf("media type %q: %w", name, err)
		}
	}
	return nil
}

func validateEncodingMetadata(meta map[string][]string) error {
	root := new(encodingMetadataNode)
	for _, key := range slices.Sorted(maps.Keys(meta)) {
		if len(meta[key]) == 0 {
			continue
		}
		path := strings.Split(strings.TrimPrefix(key, "openapi:"), ":")
		node := root
		for len(path) > 1 {
			width := 1
			switch path[0] {
			case "encoding":
				if len(path) < 3 {
					path = nil
					continue
				}
				node.named = true
				width = 2
			case "prefixEncoding":
				if len(path) < 3 {
					path = nil
					continue
				}
				index, err := strconv.Atoi(path[1])
				if err != nil || index < 0 {
					path = nil
					continue
				}
				path[1] = strconv.Itoa(index)
				node.positional = true
				width = 2
			case "itemEncoding":
				node.positional = true
			default:
				path = nil
				continue
			}
			if node.named && node.positional {
				return fmt.Errorf("%s: encoding cannot coexist with prefixEncoding or itemEncoding", key)
			}
			child := strings.Join(path[:width], ":")
			if node.children == nil {
				node.children = make(map[string]*encodingMetadataNode)
			}
			if node.children[child] == nil {
				node.children[child] = new(encodingMetadataNode)
			}
			node = node.children[child]
			path = path[width:]
		}
	}
	return nil
}
