package representation

import (
	"errors"
	"fmt"

	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/internal/examplegen"
)

// PrepareServiceExamples selects each service-owned representative once. It is
// an explicit example stage: schema-only preparation never calls it.
func PrepareServiceExamples(service *transportir.Service, source *expr.HTTPServiceExpr, generator *expr.ExampleGenerator) error {
	if service == nil || source == nil {
		return nil
	}
	for _, endpoint := range service.Endpoints {
		if endpoint.Request == nil || endpoint.Response == nil {
			return fmt.Errorf("HTTP example preparation has incomplete endpoint %s.%s", source.Name(), endpoint.MethodName)
		}
		method := source.ServiceExpr.Method(endpoint.MethodName)
		if method == nil {
			return fmt.Errorf("HTTP example preparation has no service method %s.%s", source.Name(), endpoint.MethodName)
		}
		parts := []string{"service", source.Name(), "method", method.Name}
		prepareValueExample(endpoint.Request.BodyValue, method.Payload, examplegen.ForScope(generator, append(parts, "payload")...))
		prepareValueExample(endpoint.Request.StreamingValue, method.StreamingPayload,
			examplegen.ForScope(generator, append(parts, "streaming-payload")...))
		for _, response := range append(append([]*transportir.ResponseStatus(nil), endpoint.Response.Responses...), endpoint.Response.ErrorResponses...) {
			attribute := method.Result
			role := []string{"result"}
			if response.Error != nil {
				attribute = response.Error.Attribute
				role = []string{"error", response.Error.Name}
			}
			prepareValueExample(response.BodyValue, attribute, examplegen.ForScope(generator, append(parts, role...)...))
		}
		if endpoint.Stream != nil {
			prepareValueExample(endpoint.Stream.ResponseValue, method.StreamingResult,
				examplegen.ForScope(generator, append(parts, "streaming-result")...))
		}
	}
	return nil
}

func prepareValueExample(target *transportir.ValueTarget, attribute *expr.AttributeExpr, generator *expr.ExampleGenerator) {
	if target == nil || target.Anchor == nil || target.Anchor.Example.Outcome() != 0 {
		return
	}
	prepareValueData(target.Anchor, attribute, generator)
}

func prepareValueData(data *service.ValueData, attribute *expr.AttributeExpr, generator *expr.ExampleGenerator) {
	if data == nil || data.Example.Outcome() != 0 {
		return
	}
	policy := expr.ExamplePolicy{Reachable: true, SuppressGenerated: suppressGeneratedExample(attribute)}
	selection := data.Context.SelectExample(data.Occurrence, policy)
	if source, found := selection.Source(); found {
		data.Example = data.Context.Resolve(data.Occurrence, source, expr.ValueRoleExample)
		return
	}
	data.Example = data.Context.Synthesize(selection, generator)
}

func suppressGeneratedExample(attribute *expr.AttributeExpr) bool {
	if attribute == nil {
		return false
	}
	for _, key := range []string{"openapi:generate", "openapi:example"} {
		if value, found := attribute.Meta.Last(key); found && value == "false" {
			return true
		}
	}
	return false
}

// PrepareStandaloneExamples captures one documentation occurrence for a
// schema analyzed outside a service method. It selects or synthesizes once and
// builds separate structural and projecting plans over the same owner.
func PrepareStandaloneExamples(
	attribute *expr.AttributeExpr,
	generator *expr.ExampleGenerator,
) (*transportir.ValueTarget, error) {
	if attribute == nil || attribute.Type == expr.Empty {
		return nil, nil
	}
	context := expr.NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	if err != nil {
		return nil, err
	}
	data := &service.ValueData{Context: context, Occurrence: occurrence}
	schema, err := context.NewValuePlan(occurrence, expr.ValuePlanRequest{
		Target: attribute, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanSchema,
	})
	if err != nil {
		return nil, err
	}
	documentation, err := context.NewValuePlan(occurrence, expr.ValuePlanRequest{
		Target: attribute, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanDocumentation,
	})
	if err != nil {
		return nil, err
	}
	target := &transportir.ValueTarget{
		Source: data, Anchor: data, Codec: expr.ValueCodecJSON, Plan: schema,
		AnchorPlan: documentation, ExampleOccurrence: occurrence, ExamplePlan: documentation,
	}
	if err := PrepareTargetExamples(target, attribute, generator); err != nil {
		return nil, err
	}
	return target, nil
}

// PrepareTargetExamples binds one target's selected authored group, or its
// retained representative, to exact semantic-owner plans. It never reads raw
// example values or selects a union branch in the documentation consumer.
func PrepareTargetExamples(
	target *transportir.ValueTarget,
	attribute *expr.AttributeExpr,
	generator *expr.ExampleGenerator,
) error {
	if target == nil || target.Representative != nil || len(target.Examples) > 0 {
		return nil
	}
	if target.Error != nil {
		return target.Error
	}
	if target.ExamplesPrepared {
		return nil
	}
	target.ExamplesPrepared = true
	if target.Source == nil || target.Anchor == nil || target.ExampleOccurrence.ID() == (expr.ValueIdentity{}) {
		return fmt.Errorf("HTTP example target has no captured semantic occurrence")
	}
	if !target.Plan.Root().Valid() || !target.ExamplePlan.Root().Valid() || !target.AnchorPlan.Root().Valid() {
		return fmt.Errorf("HTTP example target has no captured schema and example plans")
	}
	anchor, err := selectedValueOccurrence(target.Anchor.Occurrence, target.Selection)
	if err != nil {
		return err
	}
	anchorProjection := target.AnchorPlan
	if len(target.Selection) > 0 {
		anchorProjection, err = target.AnchorPlan.ForOccurrence(anchor, target.AnchorPlan.Root())
		if err != nil {
			return fmt.Errorf("associate selected anchor occurrence: %w", err)
		}
	}
	position := targetExamplePosition{
		schema: target.Plan.Root(), examplePlan: target.ExamplePlan.Root(), anchorPlan: target.AnchorPlan.Root(),
		exampleProjection: target.ExamplePlan, anchorProjection: anchorProjection,
		example: target.ExampleOccurrence, anchor: anchor,
	}
	target.ExampleSets = make(map[expr.ValuePlanNode]*transportir.ValueExampleSet)
	if err := prepareTargetExamplePosition(target, position, attribute, generator,
		make(map[expr.ValuePlanNode]bool), nil); err != nil {
		return err
	}
	root := target.ExampleSets[target.Plan.Root()]
	if root == nil {
		return fmt.Errorf("HTTP example target did not prepare its schema root")
	}
	target.Examples = root.Examples
	target.Representative = root.Representative
	return nil
}

type targetExamplePosition struct {
	schema            expr.ValuePlanNode
	examplePlan       expr.ValuePlanNode
	anchorPlan        expr.ValuePlanNode
	exampleProjection expr.ValuePlan
	anchorProjection  expr.ValuePlan
	example           expr.ValueOccurrence
	anchor            expr.ValueOccurrence
}

func prepareTargetExamplePosition(
	target *transportir.ValueTarget,
	position targetExamplePosition,
	attribute *expr.AttributeExpr,
	generator *expr.ExampleGenerator,
	visited map[expr.ValuePlanNode]bool,
	path []string,
) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("prepare HTTP example position %v: %w", path, err)
		}
	}()
	if !position.schema.Valid() || visited[position.schema] {
		return nil
	}
	visited[position.schema] = true
	set, err := prepareTargetExampleSet(target, position, attribute,
		examplegen.ForScope(generator, append([]string{"occurrence"}, path...)...))
	if err != nil {
		return err
	}
	target.ExampleSets[position.schema] = set

	schemaUnderlying := position.schema.Underlying()
	if schemaUnderlying.Valid() {
		next, nextErr := underlyingExamplePosition(position)
		if nextErr != nil {
			return nextErr
		}
		next.schema = schemaUnderlying
		return prepareTargetExamplePosition(target, next, schemaUnderlying.Attribute(), generator,
			visited, appendExamplePath(path, "underlying"))
	}
	if err := prepareTargetExampleMembers(target, position, generator, visited, path); err != nil {
		return err
	}
	if err := prepareTargetExampleChild(target, position.schema.Element(), position.examplePlan.Element(),
		position.anchorPlan.Element(), position.example.Element(), position.anchor.Element(), generator, visited,
		appendExamplePath(path, "element")); err != nil {
		return err
	}
	if err := prepareTargetExampleChild(target, position.schema.Key(), position.examplePlan.Key(),
		position.anchorPlan.Key(), position.example.Key(), position.anchor.Key(), generator, visited,
		appendExamplePath(path, "key")); err != nil {
		return err
	}
	return prepareTargetExampleBranches(target, position, generator, visited, path)
}

func prepareTargetExampleSet(
	target *transportir.ValueTarget,
	position targetExamplePosition,
	attribute *expr.AttributeExpr,
	generator *expr.ExampleGenerator,
) (*transportir.ValueExampleSet, error) {
	set := &transportir.ValueExampleSet{Prepared: true}
	selection := target.Source.Context.SelectExample(position.example, expr.ExamplePolicy{
		Reachable: true, SuppressGenerated: suppressGeneratedExample(attribute),
	})
	entries := selection.Entries()
	if len(entries) == 0 {
		data := target.Source
		plan := position.exampleProjection
		if position.schema == target.Plan.Root() {
			if data == target.Anchor {
				plan = target.AnchorPlan
			}
		} else {
			data = &service.ValueData{Context: target.Source.Context, Occurrence: position.example}
		}
		prepareValueData(data, attribute, generator)
		set.Representative = &transportir.ValueExample{Source: data, Plan: plan}
		return set, nil
	}
	for _, entry := range entries {
		owner, plan, err := targetExampleOwner(target, position, entry.Source())
		if err != nil {
			return nil, err
		}
		result := owner.Context.Resolve(owner.Occurrence, entry.Source(), expr.ValueRoleExample)
		set.Examples = append(set.Examples, transportir.ValueExample{
			Source:   &service.ValueData{Context: owner.Context, Occurrence: owner.Occurrence, Example: result},
			Authored: entry.Source(), Plan: plan, Summary: entry.Summary(),
			Description: entry.Description(), Meta: entry.Meta(),
		})
	}
	set.Representative = &set.Examples[len(set.Examples)-1]
	return set, nil
}

func targetExampleOwner(
	target *transportir.ValueTarget,
	position targetExamplePosition,
	source expr.ValueSource,
) (*service.ValueData, expr.ValuePlan, error) {
	type candidate struct {
		data *service.ValueData
		plan expr.ValuePlan
	}
	candidates := []candidate{{target.Anchor, target.AnchorPlan}}
	if target.Anchor != nil {
		candidates = append(candidates, candidate{
			&service.ValueData{Context: target.Anchor.Context, Occurrence: position.anchor}, position.anchorProjection,
		})
	}
	if position.example.ID() != (expr.ValueIdentity{}) {
		candidates = append(candidates, candidate{
			&service.ValueData{Context: target.Source.Context, Occurrence: position.example}, position.exampleProjection,
		})
	}
	seen := make(map[expr.ValueIdentity]bool)
	for _, candidate := range candidates {
		identity := candidate.data.Occurrence.ID()
		if seen[identity] {
			continue
		}
		seen[identity] = true
		for _, entry := range candidate.data.Context.SelectExample(candidate.data.Occurrence, expr.ExamplePolicy{Reachable: true}).Entries() {
			if entry.Source().ID() == source.ID() {
				return candidate.data, candidate.plan, nil
			}
		}
	}
	return nil, expr.ValuePlan{}, fmt.Errorf("HTTP example source has no exact semantic target owner")
}

func underlyingExamplePosition(position targetExamplePosition) (targetExamplePosition, error) {
	next := position
	next.examplePlan = position.examplePlan.Underlying()
	next.anchorPlan = position.anchorPlan.Underlying()
	if !next.examplePlan.Valid() || !next.anchorPlan.Valid() {
		return targetExamplePosition{}, fmt.Errorf("HTTP example plan does not match schema alias")
	}
	if !position.examplePlan.UnderlyingReusesSource() {
		occurrence, projection, err := associatedUnderlyingProjection(
			position.exampleProjection, position.example, next.examplePlan,
		)
		if err != nil {
			return targetExamplePosition{}, fmt.Errorf("associate example underlying occurrence: %w", err)
		}
		next.example = occurrence
		next.exampleProjection = projection
	}
	if !position.anchorPlan.UnderlyingReusesSource() {
		occurrence, projection, err := associatedUnderlyingProjection(
			position.anchorProjection, position.anchor, next.anchorPlan,
		)
		if err != nil {
			return targetExamplePosition{}, fmt.Errorf("associate anchor underlying occurrence: %w", err)
		}
		next.anchor = occurrence
		next.anchorProjection = projection
	}
	if next.example.ID() == (expr.ValueIdentity{}) || next.anchor.ID() == (expr.ValueIdentity{}) {
		return targetExamplePosition{}, fmt.Errorf("HTTP example occurrence does not match schema alias")
	}
	return next, nil
}

func associatedUnderlyingProjection(
	plan expr.ValuePlan,
	root expr.ValueOccurrence,
	target expr.ValuePlanNode,
) (expr.ValueOccurrence, expr.ValuePlan, error) {
	visited := make(map[expr.ValueIdentity]bool)
	for candidate := root.Underlying(); candidate.ID() != (expr.ValueIdentity{}) && !visited[candidate.ID()]; candidate = candidate.Underlying() {
		visited[candidate.ID()] = true
		associated, err := plan.ForOccurrence(candidate, target)
		if err == nil && associated.Root() == target {
			return candidate, associated, nil
		}
		if err != nil && !errors.Is(err, expr.ErrValuePlanAssociationNotFound) {
			return expr.ValueOccurrence{}, expr.ValuePlan{}, err
		}
	}
	return expr.ValueOccurrence{}, expr.ValuePlan{}, fmt.Errorf("value plan has no exact underlying occurrence association")
}

func prepareTargetExampleMembers(
	target *transportir.ValueTarget,
	position targetExamplePosition,
	generator *expr.ExampleGenerator,
	visited map[expr.ValuePlanNode]bool,
	path []string,
) error {
	children := make([]targetExampleNamedChild, 0, len(position.schema.Members()))
	for _, schemaMember := range position.schema.Members() {
		children = append(children, targetExampleNamedChild{name: schemaMember.Name, schema: schemaMember.Node})
	}
	return prepareTargetExampleNamedChildren(target, position, generator, visited, path, "member", children)
}

func prepareTargetExampleBranches(
	target *transportir.ValueTarget,
	position targetExamplePosition,
	generator *expr.ExampleGenerator,
	visited map[expr.ValuePlanNode]bool,
	path []string,
) error {
	children := make([]targetExampleNamedChild, 0, len(position.schema.Branches()))
	for _, schemaBranch := range position.schema.Branches() {
		children = append(children, targetExampleNamedChild{name: schemaBranch.Tag, schema: schemaBranch.Node})
	}
	return prepareTargetExampleNamedChildren(target, position, generator, visited, path, "branch", children)
}

type targetExampleNamedChild struct {
	name   string
	schema expr.ValuePlanNode
}

func prepareTargetExampleNamedChildren(
	target *transportir.ValueTarget,
	position targetExamplePosition,
	generator *expr.ExampleGenerator,
	visited map[expr.ValuePlanNode]bool,
	path []string,
	kind string,
	children []targetExampleNamedChild,
) error {
	for _, child := range children {
		examplePlan, anchorPlan, example, anchor, err := targetExampleNamedChildBindings(target, position, kind, child.name)
		if err != nil {
			return err
		}
		if err := prepareTargetExampleChild(target, child.schema, examplePlan, anchorPlan,
			example, anchor, generator, visited, appendExamplePath(path, kind, child.name)); err != nil {
			return err
		}
	}
	return nil
}

func targetExampleNamedChildBindings(
	target *transportir.ValueTarget,
	position targetExamplePosition,
	kind, name string,
) (expr.ValuePlanNode, expr.ValuePlanNode, expr.ValueOccurrence, expr.ValueOccurrence, error) {
	var examplePlan, anchorPlan expr.ValuePlanNode
	var example, anchor expr.ValueOccurrence
	var err error
	switch kind {
	case "member":
		var found bool
		examplePlan, found = valuePlanMember(position.examplePlan, name)
		if !found {
			return expr.ValuePlanNode{}, expr.ValuePlanNode{}, expr.ValueOccurrence{}, expr.ValueOccurrence{},
				fmt.Errorf("HTTP example plan has no target member %q", name)
		}
		anchorPlan, found = valuePlanMember(position.anchorPlan, name)
		if !found {
			return expr.ValuePlanNode{}, expr.ValuePlanNode{}, expr.ValueOccurrence{}, expr.ValueOccurrence{},
				fmt.Errorf("HTTP anchor plan has no target member %q", name)
		}
		example, err = associatedOccurrence(target.ExamplePlan, position.example, position.example.Members(), examplePlan)
		if err == nil {
			anchor, err = associatedOccurrence(target.AnchorPlan, position.anchor, position.anchor.Members(), anchorPlan)
		}
	case "branch":
		var found bool
		examplePlan, found = valuePlanBranch(position.examplePlan, name)
		if !found {
			return expr.ValuePlanNode{}, expr.ValuePlanNode{}, expr.ValueOccurrence{}, expr.ValueOccurrence{},
				fmt.Errorf("HTTP example plan has no target branch %q", name)
		}
		anchorPlan, found = valuePlanBranch(position.anchorPlan, name)
		if !found {
			return expr.ValuePlanNode{}, expr.ValuePlanNode{}, expr.ValueOccurrence{}, expr.ValueOccurrence{},
				fmt.Errorf("HTTP anchor plan has no target branch %q", name)
		}
		example, err = associatedBranchOccurrence(target.ExamplePlan, position.example, position.example.Branches(), examplePlan)
		if err == nil {
			anchor, err = associatedBranchOccurrence(target.AnchorPlan, position.anchor, position.anchor.Branches(), anchorPlan)
		}
	default:
		return expr.ValuePlanNode{}, expr.ValuePlanNode{}, expr.ValueOccurrence{}, expr.ValueOccurrence{},
			fmt.Errorf("HTTP example has unsupported child kind %q", kind)
	}
	if err != nil {
		return expr.ValuePlanNode{}, expr.ValuePlanNode{}, expr.ValueOccurrence{}, expr.ValueOccurrence{},
			err
	}
	return examplePlan, anchorPlan, example, anchor, nil
}

func prepareTargetExampleChild(
	target *transportir.ValueTarget,
	schema, examplePlan, anchorPlan expr.ValuePlanNode,
	example, anchor expr.ValueOccurrence,
	generator *expr.ExampleGenerator,
	visited map[expr.ValuePlanNode]bool,
	path []string,
) error {
	if !schema.Valid() {
		return nil
	}
	if !examplePlan.Valid() || !anchorPlan.Valid() || example.ID() == (expr.ValueIdentity{}) || anchor.ID() == (expr.ValueIdentity{}) {
		return fmt.Errorf("HTTP example child has no matching semantic owner")
	}
	exampleProjection, err := target.ExamplePlan.ForOccurrence(example, examplePlan)
	if err != nil {
		return err
	}
	anchorProjection, err := target.AnchorPlan.ForOccurrence(anchor, anchorPlan)
	if err != nil {
		return err
	}
	return prepareTargetExamplePosition(target, targetExamplePosition{
		schema: schema, examplePlan: examplePlan, anchorPlan: anchorPlan,
		exampleProjection: exampleProjection, anchorProjection: anchorProjection,
		example: example, anchor: anchor,
	}, schema.Attribute(), generator, visited, path)
}

func valuePlanMember(plan expr.ValuePlanNode, name string) (expr.ValuePlanNode, bool) {
	for _, member := range plan.Members() {
		if member.Name == name {
			return member.Node, true
		}
	}
	return expr.ValuePlanNode{}, false
}

func valuePlanBranch(plan expr.ValuePlanNode, tag string) (expr.ValuePlanNode, bool) {
	for _, branch := range plan.Branches() {
		if branch.Tag == tag {
			return branch.Node, true
		}
	}
	return expr.ValuePlanNode{}, false
}

func appendExamplePath(path []string, parts ...string) []string {
	return append(append([]string(nil), path...), parts...)
}

func associatedOccurrence(
	plan expr.ValuePlan,
	root expr.ValueOccurrence,
	candidates []expr.ValueMember,
	target expr.ValuePlanNode,
) (expr.ValueOccurrence, error) {
	values := make([]expr.ValueOccurrence, 0, len(candidates))
	for _, candidate := range candidates {
		values = append(values, candidate.Occurrence)
	}
	return associatedDescendantOccurrence(plan, root, values, target, "member")
}

func associatedBranchOccurrence(
	plan expr.ValuePlan,
	root expr.ValueOccurrence,
	candidates []expr.ValueBranch,
	target expr.ValuePlanNode,
) (expr.ValueOccurrence, error) {
	values := make([]expr.ValueOccurrence, 0, len(candidates))
	for _, candidate := range candidates {
		values = append(values, candidate.Occurrence)
	}
	return associatedDescendantOccurrence(plan, root, values, target, "branch")
}

func associatedDescendantOccurrence(
	plan expr.ValuePlan,
	root expr.ValueOccurrence,
	candidates []expr.ValueOccurrence,
	target expr.ValuePlanNode,
	kind string,
) (expr.ValueOccurrence, error) {
	queue := append(append([]expr.ValueOccurrence(nil), candidates...), root)
	visited := make(map[expr.ValueIdentity]bool)
	var found expr.ValueOccurrence
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		identity := current.ID()
		if identity == (expr.ValueIdentity{}) || visited[identity] {
			continue
		}
		visited[identity] = true
		associated, err := plan.ForOccurrence(current, target)
		if err == nil && associated.Root() == target {
			if found.ID() != (expr.ValueIdentity{}) && found.ID() != identity {
				return expr.ValueOccurrence{}, fmt.Errorf("HTTP example %s representation is ambiguous", kind)
			}
			found = current
		}
		if err != nil && !errors.Is(err, expr.ErrValuePlanAssociationNotFound) {
			return expr.ValueOccurrence{}, err
		}
		queue = append(queue, current.Underlying(), current.Element(), current.Key())
		for _, member := range current.Members() {
			queue = append(queue, member.Occurrence)
		}
		for _, branch := range current.Branches() {
			queue = append(queue, branch.Occurrence)
		}
	}
	if found.ID() == (expr.ValueIdentity{}) {
		return expr.ValueOccurrence{}, fmt.Errorf("HTTP example %s has no associated occurrence", kind)
	}
	return found, nil
}

func selectedValueOccurrence(root expr.ValueOccurrence, selection []string) (expr.ValueOccurrence, error) {
	current := root
	for _, name := range selection {
		var found expr.ValueOccurrence
		for _, member := range current.Members() {
			if member.Name == name || expr.AttributeName(member.Name) == expr.AttributeName(name) {
				if found.ID() != (expr.ValueIdentity{}) {
					return expr.ValueOccurrence{}, fmt.Errorf("HTTP example selection %q is ambiguous", name)
				}
				found = member.Occurrence
			}
		}
		if found.ID() == (expr.ValueIdentity{}) {
			return expr.ValueOccurrence{}, fmt.Errorf("HTTP example selection has no member %q", name)
		}
		current = found
	}
	return current, nil
}
