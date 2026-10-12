package testdata

// RecursiveProjectionServiceHarness supplies canonical values with excluded and selected secrets.
const RecursiveProjectionServiceHarness = `package projection_test

import (
	"context"
	svc "example.com/projection/gen/projection"
	"testing"
)

type implementation struct{ secret string }

func (s implementation) Show(context.Context) (*svc.Wrap, error) {
	owner := func() *svc.Owner {
		return &svc.Owner{ID: "owner", Secret: s.secret}
	}
	choice := svc.NewChoiceOwner(owner())
	return &svc.Wrap{ID: "wrap", Owner: owner(), Owners: []*svc.Owner{owner()}, Items: []*svc.Owner{owner()}, ByKey: map[string]*svc.Owner{"k": owner()}, Grouped: map[string][]*svc.Owner{"k": {owner()}}, Pick: &svc.Pick{Choice: &choice}, Holder: &svc.Holder{Owner: owner()}, FullOwner: &svc.Owner{ID: "full", Secret: "allowed"}}, nil
}

func TestSelectedProjection(t *testing.T) {
	value, err := (implementation{secret: "excluded"}).Show(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	selected := svc.ProjectWrap(value)
	for name, secret := range map[string]*string{
		"direct":     selected.Owner.Secret,
		"collection": selected.Owners[0].Secret,
		"array":      selected.Items[0].Secret,
		"map":        selected.ByKey["k"].Secret,
		"map_array":  selected.Grouped["k"][0].Secret,
		"wrapper":    selected.Holder.Owner.Secret,
	} {
		if secret != nil {
			t.Errorf("%s copied excluded field %q", name, *secret)
		}
	}
	branch, ok := selected.Pick.Choice.AsOwner()
	if !ok || branch.Secret != nil {
		t.Errorf("union copied excluded field: %+v", branch)
	}
	if selected.FullOwner.Secret == nil || *selected.FullOwner.Secret != "allowed" {
		t.Errorf("explicit child selection lost field")
	}
	if svc.ProjectWrap(nil) != nil {
		t.Errorf("nil root became present")
	}
	nilCollections := svc.ProjectWrap(&svc.Wrap{})
	if nilCollections.Items != nil || nilCollections.ByKey != nil {
		t.Errorf("absent collections became present")
	}
	emptyCollections := svc.ProjectWrap(&svc.Wrap{Items: []*svc.Owner{}, ByKey: map[string]*svc.Owner{}})
	if emptyCollections.Items == nil || len(emptyCollections.Items) != 0 || emptyCollections.ByKey == nil || len(emptyCollections.ByKey) != 0 {
		t.Errorf("empty collections not preserved")
	}
}
`
