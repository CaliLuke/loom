package openapiimport

import "strings"

func (r *renderer) importedTags() []Tag {
	seen := make(map[string]struct{})
	tags := make([]Tag, 0, len(r.document.Tags))
	add := func(tag Tag) {
		if _, ok := seen[tag.Name]; ok {
			return
		}
		seen[tag.Name] = struct{}{}
		tags = append(tags, tag)
	}
	for _, tag := range r.document.TagMetadata {
		add(tag)
	}
	for _, name := range r.document.Tags {
		add(Tag{Name: name})
	}
	for _, operation := range r.document.Operations {
		for _, name := range operation.Tags {
			add(Tag{Name: name})
		}
	}
	return tags
}

func (r *renderer) tagMetadata(tag Tag) error {
	prefix := "openapi:tag:" + tag.Name
	r.line("Meta(%q)", prefix)
	fields := []renderedMetadata{
		{name: "summary", value: tag.Summary},
		{name: "desc", value: tag.Description},
		{name: "parent", value: tag.Parent},
		{name: "kind", value: tag.Kind},
		{name: "url", value: tag.ExternalDocsURL},
		{name: "url:desc", value: tag.ExternalDocsDescription},
	}
	for _, field := range fields {
		if field.value != "" {
			r.line("Meta(%q, %q)", prefix+":"+field.name, field.value)
		}
	}
	extensions, err := renderedExtensions("", tag.Extensions)
	if err != nil {
		return err
	}
	for _, extension := range extensions {
		name := strings.TrimPrefix(extension.name, "openapi:extension:")
		r.line("Meta(%q, %q)", prefix+":extension:"+name, extension.value)
	}
	return nil
}
