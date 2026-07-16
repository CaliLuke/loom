package codegen_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/eval"
)

func TestRegistryPublicExtensionContract(t *testing.T) {
	registry := codegen.NewRegistry()
	var calls []string
	registry.RegisterPluginFirst("extension", "gen", func(string, []eval.Root) error {
		calls = append(calls, "prepare")
		return nil
	}, func(_ string, _ []eval.Root, files []*codegen.File) ([]*codegen.File, error) {
		calls = append(calls, "generate")
		return append(files, &codegen.File{Path: "gen/extension.go"}), nil
	})

	require.NoError(t, registry.RunPluginsPrepare("gen", "example.com/service/gen", nil))
	files, err := registry.RunPlugins("gen", "example.com/service/gen", nil, nil)

	require.NoError(t, err)
	require.Equal(t, []string{"prepare", "generate"}, calls)
	require.Len(t, files, 1)
	require.Equal(t, "gen/extension.go", files[0].Path)
}

func TestTemplateSectionsPublicExtensionContract(t *testing.T) {
	file := &codegen.File{SectionTemplates: []*codegen.SectionTemplate{{
		Name:   "external-type",
		Source: "type External struct{}\n",
	}}}

	sections := file.AllSections()
	require.Len(t, sections, 1)
	require.Equal(t, "external-type", sections[0].SectionName())
}
