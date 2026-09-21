package service

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"path/filepath"
	"sort"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type (
	authorizationManifestData struct {
		Version int                           `json:"version"`
		Service string                        `json:"service"`
		Methods []authorizationManifestMethod `json:"methods"`
	}
	authorizationManifestMethod struct {
		Method         string                              `json:"method"`
		Phase          string                              `json:"phase"`
		Classification authorizationManifestClassification `json:"classification"`
	}
	authorizationManifestClassification struct {
		Exemption    string                             `json:"exemption,omitempty"`
		Selector     *string                            `json:"selector,omitempty"`
		Cases        []authorizationManifestCase        `json:"cases,omitempty"`
		Requirements []authorizationManifestRequirement `json:"requirements,omitempty"`
	}
	authorizationManifestCase struct {
		Value          string                              `json:"value"`
		Classification authorizationManifestClassification `json:"classification"`
	}
	authorizationManifestRequirement struct {
		Name      string                         `json:"name"`
		InputType string                         `json:"input_type"`
		Bindings  []authorizationManifestBinding `json:"bindings"`
	}
	authorizationManifestBinding struct {
		Input   string `json:"input"`
		Payload string `json:"payload"`
	}
)

func authorizationManifest(svc *Data) *codegen.File {
	manifest := authorizationManifestData{Version: 1, Service: svc.Name}
	for name, m := range svc.Authorization.methods {
		manifest.Methods = append(manifest.Methods, authorizationManifestMethod{
			Method: name, Phase: "invocation", Classification: authorizationManifestClass(m.expr.Authorization),
		})
	}
	sort.Slice(manifest.Methods, func(i, j int) bool {
		return manifest.Methods[i].Method < manifest.Methods[j].Method
	})
	content, err := json.Marshal(manifest, json.Deterministic(true), jsontext.WithIndent("  "))
	if err != nil {
		panic(err)
	}
	return &codegen.File{
		Path:     filepath.Join(codegen.Gendir, svc.PathName, "authorization.json"),
		Sections: []codegen.Section{codegen.NewRawSection("authorization-manifest", string(content))},
	}
}

func authorizationManifestClass(a *expr.MethodAuthorizationExpr) authorizationManifestClassification {
	c := authorizationManifestClassification{Exemption: a.Exemption}
	if a.VariantSelection {
		c.Selector = &a.Selector
	}
	for _, use := range a.Requirements {
		r := authorizationManifestRequirement{Name: use.Requirement.Name, InputType: use.Requirement.Input.Type.Name(), Bindings: []authorizationManifestBinding{}}
		for _, binding := range use.Bindings {
			r.Bindings = append(r.Bindings, authorizationManifestBinding{Input: binding.Input, Payload: binding.Payload})
		}
		sort.Slice(r.Bindings, func(i, j int) bool {
			return r.Bindings[i].Input < r.Bindings[j].Input
		})
		c.Requirements = append(c.Requirements, r)
	}
	for _, branch := range a.Cases {
		c.Cases = append(c.Cases, authorizationManifestCase{Value: branch.Value, Classification: authorizationManifestClass(branch.Authorization)})
	}
	sort.Slice(c.Cases, func(i, j int) bool {
		return c.Cases[i].Value < c.Cases[j].Value
	})
	return c
}
