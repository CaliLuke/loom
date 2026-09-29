import ValueContract.GraphValidation
import ValueContract.MapKeys

namespace ValueContract.Candidate

/-- Independent raw binary meaning: nil remains nil, while each original
native byte contributes exactly one octet irrespective of container naming. -/
def NativeByteValue (sequence : NativeByteSequence) (value : Value) : Prop :=
  (NativeBytesNil sequence ∧ value = .nilBytes) ∨
  (¬ NativeBytesNil sequence ∧ ∃ bytes, NativeByteOctets sequence bytes ∧
    value = .scalar (.bytes bytes))

theorem nativeByteValue_iff (sequence : NativeByteSequence) (value : Value) :
    sequence.rawValue = value ↔ NativeByteValue sequence value := by
  simp only [NativeByteSequence.rawValue, NativeByteValue,
    ← nativeBytesNil_iff, ← nativeByteOctets_iff]
  cases sequence.isNil <;> simp [eq_comm]

/-- Independent extraction preserves nil and each original native-byte host
class. This relation does not inspect a declaration, branch or target. -/
def ArrayInputEntries (input : Input) (items : Option (List Input)) : Prop :=
  match input with
  | .nilArray => items = none
  | .array original => items = some original
  | .byteSequence sequence =>
    (NativeBytesNil sequence ∧ items = none) ∨
    (¬ NativeBytesNil sequence ∧ items = some sequence.inputs)
  | _ => False

/-- Raw source interpretation is structural and preserves every scalar kind,
name and nil marker. Key spelling is a narrow codec relation; no resolver
success premise, target representation, or DSL alias lookup occurs here. -/
def RawResolutionAt (keys : KeyCodec) : Nat → Input → Value → Prop
  | 0, _, _ => False
  | depth + 1, input, value => match input, value with
    | .host identity payload, .host actual child =>
        identity = actual ∧ RawResolutionAt keys depth payload child
    | .null, .null | .nilBytes, .nilBytes | .nilArray, .nilArray | .nilMap, .nilMap => True
    | .scalar original, .scalar actual => original.value = actual
    | .byteSequence sequence, value => NativeByteValue sequence value
    | .array inputs, .array values => All₂ (RawResolutionAt keys depth) inputs values
    | .object inputs, .object fields values => fields = [] ∧
        (inputs.map Prod.fst).Nodup ∧
        All₂ (fun original actual => original.1 = actual.1 ∧
          RawResolutionAt keys depth original.2 actual.2) inputs values
    | .map inputs, .map values =>
        (∃ names, names.Nodup ∧ All₂ (KeySpelling keys) (inputs.map (fun entry => entry.1.value)) names) ∧
        All₂ (fun original actual => original.1.value = actual.1 ∧
          RawResolutionAt keys depth original.2 actual.2) inputs values
    | _, _ => False

def RawResolution (keys : KeyCodec) (input : Input) (value : Value) : Prop :=
  ∃ depth, RawResolutionAt keys depth input value

/-- The chosen named input is supplied and all earlier entries fail that
same name/presence test. This positional relation does not call a lookup. -/
def FirstSupplied (name : String) (entries : List (String × Input))
    (input : Input) : Prop :=
  input ≠ .absent ∧ ∃ before after,
    entries = before ++ (name, input) :: after ∧
    ∀ entry ∈ before, entry.1 ≠ name ∨ entry.2 = .absent

def NoSupplied (name : String) (entries : List (String × Input)) : Prop :=
  ∀ entry ∈ entries, entry.1 = name → entry.2 = .absent

/-- Wire spelling wins only after every supplied spelling has been validated.
A missing slot is the absence of both spellings, never an explicit null. -/
def MemberChoice (member : Member) (entries : List (String × Input)) : Option Input → Prop
  | some input => FirstSupplied member.wireAlias entries input ∨
      (NoSupplied member.wireAlias entries ∧ FirstSupplied member.sourceName entries input)
  | none => NoSupplied member.wireAlias entries ∧ NoSupplied member.sourceName entries

/-- A member derivation consumes a recursively established relation. Every
supplied alias is checked, including a losing alias and cross-member overlaps;
missing paths are retained exactly and prefixed with the member identity. -/
def MemberResolution (child : Identity → Input → Resolution → Prop)
    (complete : Bool) (member : Member) (entries : List (String × Input))
    (result : Identity × Resolution) : Prop :=
  (∀ entry ∈ entries, entry.2 ≠ .absent →
    (entry.1 = member.sourceName ∨ entry.1 = member.wireAlias) →
      ∃ value, child member.child entry.2 value) ∧
  ((MemberChoice member entries none ∧
      (complete = false ∨ member.required = false) ∧
      result = (member.identity, ⟨.absent, if member.required then [[member.identity]] else []⟩)) ∨
    (∃ input value, MemberChoice member entries (some input) ∧
      child member.child input value ∧
      result = (member.identity, {value with missing := value.missing.map (member.identity :: ·)})))

/-- Scalar source matching allows only the established built-in coercions;
raw nil Bytes are the distinct legacy normalization to an empty byte sequence. -/
def ScalarResolution (checks : ExternalScalarChecks) (keys : KeyCodec) (kind : ScalarKind) (rules : ScalarRules)
    (input : Input) (result : Resolution) : Prop :=
  ∃ scalar, ((∃ original, stripHostInput input = .scalar original ∧ SourceCoerces keys rules kind original scalar) ∨
      (stripHostInput input = .nilBytes ∧ kind = .bytes ∧ scalar = .bytes []) ∨
      (∃ sequence bytes, stripHostInput input = .byteSequence sequence ∧ kind = .bytes ∧
        NativeBytesAdmitted sequence ∧ NativeByteOctets sequence bytes ∧ scalar = .bytes bytes)) ∧
    scalarAllowed checks rules scalar = true ∧ result = ⟨.scalar scalar, []⟩

/-- Collections retain every child resolution and concatenate its missing
paths in order. Nil and empty remain different semantic source constructors. -/
def ArrayResolution (child : Input → Resolution → Prop) (bounds : LengthBounds)
    (input : Input) (result : Resolution) : Prop :=
  (ArrayInputEntries (stripHostInput input) none ∧ lengthAllowed bounds 0 = true ∧ result = ⟨.nilArray, []⟩) ∨
  (∃ inputs children, ArrayInputEntries (stripHostInput input) (some inputs) ∧ lengthAllowed bounds inputs.length = true ∧
    All₂ child inputs children ∧ result = ⟨.array (children.map Resolution.value),
      children.flatMap Resolution.missing⟩)

/-- A branch has already been checked recursively before ranking. Ambiguity
means viable but obstructed; invalid/unsupported/cyclic branches cannot be
silently treated as successful interpretations. -/
structure BranchCandidate where
  branch : Identity
  preferred : Bool
  result : ResolveResult
  deriving Repr

def BranchCandidate.viable (candidate : BranchCandidate) : Bool :=
  match candidate.result with
  | .ok _ | .error .ambiguous => true
  | _ => false

def BranchCandidate.complete (candidate : BranchCandidate) : Bool :=
  match candidate.result with
  | .ok resolution => resolution.missing.isEmpty
  | _ => false

/-- The independent ranking rule describes the retained list, not the result
of an executable selector. Multiplicity matters: two preferred candidates are
ambiguous even if their eventual projected payloads would be equal. -/
inductive RankedCandidates (candidates : List BranchCandidate) :
    Except Failure BranchCandidate → Prop where
  | uniquePreferred (chosen : BranchCandidate)
      (preferred : candidates.filter (fun c => c.viable && c.preferred) = [chosen]) :
      RankedCandidates candidates (.ok chosen)
  | soleUnpreferred (chosen : BranchCandidate)
      (preferred : candidates.filter (fun c => c.viable && c.preferred) = [])
      (viable : candidates.filter BranchCandidate.viable = [chosen]) :
      RankedCandidates candidates (.ok chosen)
  | manyPreferred
      (many : 2 ≤ (candidates.filter (fun c => c.viable && c.preferred)).length) :
      RankedCandidates candidates (.error .ambiguous)
  | manyUnpreferred
      (preferred : candidates.filter (fun c => c.viable && c.preferred) = [])
      (many : 2 ≤ (candidates.filter BranchCandidate.viable).length) :
      RankedCandidates candidates (.error .ambiguous)
  | none (empty : candidates.filter BranchCandidate.viable = []) :
      RankedCandidates candidates (.error .invalid)

/-- Complete-first eligibility is separate from preference. An ambiguous
nonempty complete set never consults the fallback set. -/
inductive CompleteFirst (complete fallback : List BranchCandidate) :
    Except Failure BranchCandidate → Prop where
  | completeSet (present : complete ≠ [])
      (ranked : RankedCandidates complete result) :
      CompleteFirst complete fallback result
  | fallbackSet (empty : complete = [])
      (ranked : RankedCandidates fallback result) :
      CompleteFirst complete fallback result

end ValueContract.Candidate
