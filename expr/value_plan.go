package expr

type (
	// ValuePlan is an immutable target representation of a semantic occurrence.
	// Its zero value is invalid. Codec ownership is independent of media labels.
	ValuePlan struct {
		context      *valueContextIdentity
		source       ValueOccurrence
		root         *valuePlanNode
		selection    []uint64
		association  *valuePlanAssociation
		associations map[*valueOccurrenceNode][]*valuePlanAssociation
		queryIndex   *valuePlanQueryIndex
		projections  *valueProjectionMemo
		// nodes indexes the complete captured root. Subplans leave it nil so
		// membership remains restricted to their own reachable subtree.
		nodes map[*valuePlanNode]bool
	}

	// ValuePlanRequest describes an already finalized effective target. It does
	// not authorize resolving a source again or selecting a different union.
	ValuePlanRequest struct {
		// Target is the finalized view, body or location attribute.
		// It is captured at this call and must be nil when TargetOccurrence is set.
		Target *AttributeExpr
		// TargetOccurrence reuses an immutable target captured in this context.
		// It preserves that snapshot even if the original attribute later changes.
		// Exactly one of Target and TargetOccurrence must be supplied.
		TargetOccurrence ValueOccurrence
		// Selection is an authored member or union-branch path from the semantic
		// root. A branch projects only values selecting that branch; other union
		// values are absent. An empty path selects the root itself.
		Selection []string
		// Codec identifies the actual encoding owner, not a documentation MIME.
		Codec ValueCodec
		// Use distinguishes runtime-backed targets from documentation alone.
		Use ValuePlanUse
		// Fields supplies the actual emitter policy for every runtime object
		// member, including hidden fields. Documentation plans may omit it.
		Fields []ValueFieldPolicy
		// Containers supplies actual runtime unknown-member policy for each
		// object or union envelope, independently of schema acceptance.
		Containers []ValueContainerPolicy
		// Codecs records node-local encoding boundaries selected by the actual
		// representation owner. Unspecified descendants inherit their parent codec.
		Codecs []ValueCodecPolicy
	}

	// ValueCodecPolicy binds an encoding owner to one exact finalized target occurrence.
	ValueCodecPolicy struct {
		// Target identifies the occurrence before capture, not a type-name lookup.
		Target *AttributeExpr
		// Codec is the actual encoding owner for this node and its descendants.
		Codec ValueCodec
	}

	// ValueFieldPolicy records the emitted representation of one target member.
	// Parent and Target are exact finalized attribute identities, before copying.
	ValueFieldPolicy struct {
		// Parent is the object attribute that owns the field.
		Parent *AttributeExpr
		// Target is the field occurrence, not its reusable declaration.
		Target *AttributeExpr
		// Name is its full authored name, disambiguating reused attribute pointers.
		Name string
		// WireName is the actual encoded member name.
		WireName string
		// Visible reports whether the target emits this member.
		Visible bool
		// Required reports whether the target contract requires the member.
		Required bool
		// Presence is the actual emitted field omission/default policy.
		Presence ValueFieldPresence
		// NumericKind is the effective scalar precision, or zero for a
		// nonnumeric member. It must agree with the finalized target type.
		NumericKind Kind
		// ImplicitDefault is the codec-owned scalar default when Presence is
		// ValueFieldImplicitDefault. Other policies must leave it nil.
		ImplicitDefault any
	}

	// ValueContainerPolicy describes the actual decoder of one target container.
	ValueContainerPolicy struct {
		// Target is the finalized object or union attribute before capture.
		Target *AttributeExpr
		// RejectUnknownMembers selects rejection instead of ignoring extra wire members.
		RejectUnknownMembers bool
		// PreserveAdditional reports whether the encoder retains undeclared semantic members.
		PreserveAdditional bool
	}

	// ValueCodec identifies an actual serialization owner.
	ValueCodec uint8
	// ValuePlanUse identifies which target guarantees a plan must establish.
	ValuePlanUse uint8
	// ValueFieldPresence identifies field-local observation and emission rules.
	ValueFieldPresence uint8

	valuePlanNode struct {
		id                  uint64
		source              *valueOccurrenceNode
		targetDeclarationID string
		attribute           *AttributeExpr
		validation          *ValidationExpr
		kind                Kind
		members             []valuePlanMember
		branches            []valuePlanBranch
		element             *valuePlanNode
		key                 *valuePlanNode
		alias               *valuePlanNode
		aliasReusesSource   bool
		codec               ValueCodec
		documentary         bool
		schemaOnly          bool
		nullable            bool
		nonNullableElements bool
		untagged            bool
		typeKey             string
		valueKey            string
		schemaUnknown       bool
		runtimeUnknown      bool
		preserveAdditional  bool
		hasEnum             bool
		enumClauses         [][]ResolvedValue
		enumPlans           []*valuePlanNode
	}

	valuePlanAssociation struct {
		id        uint64
		source    *valueOccurrenceNode
		root      *valuePlanNode
		selection []uint64
	}

	valuePlanMember struct {
		source          uint64
		name            string
		wire            string
		required        bool
		visible         bool
		presence        ValueFieldPresence
		implicitDefault any
		node            *valuePlanNode
	}

	valuePlanBranch struct {
		source uint64
		tag    string
		node   *valuePlanNode
	}
)

const (
	// ValueCodecJSON is Loom's JSON-v2 codec, including JSONValue materialization.
	ValueCodecJSON ValueCodec = iota + 1
	// ValueCodecProtoJSON is the protobuf JSON codec.
	ValueCodecProtoJSON
	// ValueCodecText is the selected text codec for a body or transport location.
	ValueCodecText
	// ValueCodecRaw is an unencoded binary body.
	ValueCodecRaw
	// ValueCodecMultipart is the multipart body encoder.
	ValueCodecMultipart
	// ValueCodecForm is the form body encoder.
	ValueCodecForm
	// ValueCodecCustom is an explicitly selected external codec boundary.
	ValueCodecCustom
)

const (
	// ValuePlanRuntime requires same-wire runtime decoding guarantees.
	ValuePlanRuntime ValuePlanUse = iota + 1
	// ValuePlanDocumentation requires schema/media validity without a decoder claim.
	ValuePlanDocumentation
	// ValuePlanSchema captures structure and owned constraint declarations only.
	// It does not resolve enum values, select examples or authorize value projection.
	ValuePlanSchema
)

const (
	// ValueFieldRetain preserves present values, including empty and null.
	ValueFieldRetain ValueFieldPresence = iota + 1
	// ValueFieldOmitEmpty omits a field empty after child observation.
	ValueFieldOmitEmpty
	// ValueFieldOmitAbsent preserves explicit null/empty and omits only absence.
	ValueFieldOmitAbsent
	// ValueFieldOmitZero omits a native zero under an authored omitzero tag.
	ValueFieldOmitZero
	// ValueFieldOmitEmptyOrZero applies both authored omission predicates.
	ValueFieldOmitEmptyOrZero
	// ValueFieldImplicitDefault follows a codec's implicit default presence.
	ValueFieldImplicitDefault
)
