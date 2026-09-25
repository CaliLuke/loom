package service

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
)

// TestServiceFilesMappedNames checks that the service and views packages name
// attributes declared with a transport element name suffix, such as "n:m",
// after the part that precedes the colon: the JSON tags, the union branch
// kinds, the types promoted from the branches of a OneOf block, the union
// type, the view attribute lists and the view validation errors.
func TestServiceFilesMappedNames(t *testing.T) {
	root := codegen.RunDSL(t, mappedNamesServiceDSL)
	services := NewServicesData(root)
	var code string
	for _, service := range root.Services {
		for _, file := range Files("gen", service, services, make(map[string][]string)) {
			code += codegen.SectionsCode(t, file.Sections[1:])
		}
		if views := ViewsFile("gen", service, services); views != nil {
			code += codegen.SectionsCode(t, views.Sections[1:])
		}
	}

	for _, want := range []string{
		"type Envelope struct {\n" +
			"\tN      *string     `json:\"n,omitempty\"`\n" +
			"\tReq    int         `json:\"req\"`\n" +
			"\tPick   IntOrString `json:\"pick\"`\n" +
			"\tObj    *Leaf       `json:\"obj\"`\n" +
			"\tChoice *Choice     `json:\"choice,omitempty\"`\n" +
			"}\n",
		"type Leaf struct {\n\tLeaf  *string `json:\"leaf,omitempty\"`\n\tCount int     `json:\"count\"`\n}\n",
		"type ChoiceText string\n",
		"type Choice struct {\n\tkind       ChoiceKind\n\tText       ChoiceText\n\tLeafBranch *Leaf\n}\n",
		"\t// ChoiceKindText identifies the text branch of the union.\n\tChoiceKindText ChoiceKind = \"text\"\n",
		"\tChoiceKindLeafBranch ChoiceKind = \"leaf_branch\"\n",
		"\t\t\"default\": {\n\t\t\t\"n\",\n\t\t\t\"req\",\n\t\t},\n\t\t\"tiny\": {\n\t\t\t\"req\",\n\t\t},\n",
		"loom.MissingFieldError(\"req\", \"result\")",
	} {
		assert.Contains(t, code, want)
	}
	assert.Empty(t, regexp.MustCompile(`"[a-z_]+:[a-z_]+`).FindAllString(code, -1), "names with the element name suffix")
	assert.NotContains(t, code, "Choice2")
}

func mappedNamesServiceDSL() {
	var Leaf = dsl.Type("Leaf", func() {
		dsl.Attribute("leaf:l", dsl.String)
		dsl.Attribute("count:c", dsl.Int)
		dsl.Required("count:c")
	})
	var Envelope = dsl.Type("Envelope", func() {
		dsl.Attribute("n:m", dsl.String, func() {
			dsl.MinLength(2)
		})
		dsl.Attribute("req:r", dsl.Int)
		dsl.Attribute("pick:p", dsl.OneOf(dsl.String, dsl.Int))
		dsl.Attribute("obj:o", Leaf)
		dsl.OneOf("choice:ch", func() {
			dsl.Attribute("text:t", dsl.String)
			dsl.Attribute("leaf_branch:lb", Leaf)
		})
		dsl.Required("req:r", "pick:p", "obj:o")
	})
	var Probe = dsl.ResultType("application/vnd.probe", "Probe", func() {
		dsl.Attributes(func() {
			dsl.Attribute("n:m", dsl.String)
			dsl.Attribute("req:r", dsl.Int)
			dsl.Required("req:r")
		})
		dsl.View("default", func() {
			dsl.Attribute("n:m")
			dsl.Attribute("req:r")
		})
		dsl.View("tiny", func() {
			dsl.Attribute("req:r")
		})
	})
	dsl.Service("mappednames", func() {
		dsl.Method("echo", func() {
			dsl.Payload(Envelope)
			dsl.Result(Envelope)
		})
		dsl.Method("show", func() {
			dsl.Result(Probe)
		})
	})
}
