import ValueContract.ResolutionSpec

namespace ValueContract.Candidate

/-- Map input admits authored maps and object-shaped string keys. Host evidence
is transparent to declared shape matching, without rewriting child inputs. -/
def MapInputEntries (input : Input) (entries : List (SourceScalar × Input)) : Prop :=
  match stripHostInput input with
  | .map original => entries = original
  | .object fields => entries = fields.map (fun field => (⟨.string field.1, .builtinString⟩, field.2))
  | _ => False

/-- Object input admits objects and maps whose every key is a string. The
pairwise relation preserves order, names and each exact supplied child. -/
def ObjectInputEntries (input : Input) (entries : List (String × Input)) : Prop :=
  match stripHostInput input with
  | .object original => entries = original
  | .map original => All₂ (fun source actual => source.1.value = .string actual.1 ∧
      source.2 = actual.2) original entries
  | _ => False

/-- A declared map key is normalized only through the supported scalar
coercions. Its child input is retained byte-for-byte in the finite model. -/
def MapKeyResolution (checks : ExternalScalarChecks) (keys : KeyCodec) (kind : MapKeyKind)
    (rules : ScalarRules) (source : SourceScalar × Input) (actual : Scalar × Input) : Prop :=
  (match kind with
  | .builtin => source.1.value = actual.1
  | .scalar expected => SourceCoerces keys rules expected source.1 actual.1) ∧
  mapKeyCompatible kind actual.1 = true ∧ scalarAllowed checks rules actual.1 = true ∧
  source.2 = actual.2

/-- Independent declared-map resolution includes key spelling and collision
checks before child derivations. Nil and nonnil empty maps remain distinct. -/
def MapResolution (checks : ExternalScalarChecks) (keys : KeyCodec)
    (kind : MapKeyKind) (rules : ScalarRules) (bounds : LengthBounds)
    (child : Input → Resolution → Prop) (input : Input) (result : Resolution) : Prop :=
  (stripHostInput input = .nilMap ∧ lengthAllowed bounds 0 = true ∧
    result = ⟨.nilMap, []⟩) ∨
  (∃ (entries : List (SourceScalar × Input)) (normalized : List (Scalar × Input)) (children : List (Scalar × Resolution)),
    MapInputEntries input entries ∧ lengthAllowed bounds entries.length = true ∧
    All₂ (MapKeyResolution checks keys kind rules) entries normalized ∧
    (∃ names, names.Nodup ∧ All₂ (KeySpelling keys) (normalized.map Prod.fst) names) ∧
    All₂ (fun source actual => source.1 = actual.1 ∧ child source.2 actual.2) normalized children ∧
    result = ⟨.map (children.map (fun entry => (entry.1, entry.2.value))),
      children.flatMap (fun entry => entry.2.missing)⟩)

/-- Additional authored entries have neither an authored nor a wire spelling
of any declared member. Cross-member spelling overlap is not forbidden. -/
def AdditionalEntries (members : List Member) (entries : List (String × Input)) :
    List (String × Input) :=
  entries.filter (fun entry => decide (∀ member ∈ members,
    entry.1 ≠ member.sourceName ∧ entry.1 ≠ member.wireAlias))

/-- Independent object-body resolution validates all supplied aliases, retains
member identities and missing paths, and validates every admitted extra value.
The raw depth is an explicit derivation index, not a fixed domain cutoff. -/
def ObjectResolution (keys : KeyCodec) (rawDepth : Nat) (complete : Bool)
    (members : List Member) (isOpen : Bool)
    (child : Identity → Input → Resolution → Prop) (input : Input) (result : Resolution) : Prop :=
  ∃ (entries : List (String × Input)) (fields : List (Identity × Resolution))
    (extras : List (String × Value)),
    ObjectInputEntries input entries ∧ (entries.map Prod.fst).Nodup ∧
    (isOpen = true ∨ AdditionalEntries members entries = []) ∧
    All₂ (fun member field => MemberResolution child complete member entries field) members fields ∧
    All₂ (fun source actual => source.1 = actual.1 ∧
      RawResolutionAt keys rawDepth source.2 actual.2) (AdditionalEntries members entries) extras ∧
    result = ⟨.object (fields.map (fun entry => (entry.1, entry.2.value))) extras,
      fields.flatMap (fun entry => entry.2.missing)⟩

end ValueContract.Candidate
