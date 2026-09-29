import ValueContract.ScalarProjection
import ValueContract.MapKeys

namespace ValueContract.Candidate

/-- JSON validity excludes protobuf-only constructors. Lexical numeric validity
is a named parsing boundary; no source numeric kind is available to this check. -/
def jsonValidAt : Nat → NumericCodec → Wire → Prop
  | 0, _, _ => False
  | depth + 1, numbers, wire => match wire with
    | .null | .boolean _ | .text _ => True
    | .number text => (numbers.schemaNumber text).isSome = true
    | .array items => All (jsonValidAt depth numbers) items
    | .object entries => (entries.map Prod.fst).Nodup ∧
      ∀ entry ∈ entries, jsonValidAt depth numbers entry.2
    | _ => False

def JSONValid (numbers : NumericCodec) (wire : Wire) : Prop :=
  ∃ depth, jsonValidAt depth numbers wire

/-- A deterministic member order is part of canonical structured materialization.
The actual JSON serializer's ordering/options are checked by correspondence tests. -/
def orderedMembers (entries : List (String × Wire)) : List (String × Wire) :=
  entries.mergeSort (fun left right => left.1 ≤ right.1)

/-- Independent meaning of materializing raw built-in Any input. Nil containers
use their actual codec meaning, not declared-collection example incompleteness.
This relation does not mention the materializer or the target projector. -/
def materializesAt : Nat → ScalarCodecs → Value → Wire → Prop
  | 0, _, _, _ => False
  | depth + 1, codecs, value, wire => match value, wire with
    | .host _ payload, wire => materializesAt depth codecs payload wire
    | .null, .null => True
    | .nilBytes, .text text => text = codecs.bytes.encode []
    | .nilArray, .array [] | .nilMap, .object [] => True
    | .scalar scalar, wire => CanonicalScalar codecs .json scalar wire ∧
      JSONValid codecs.numbers wire
    | .array values, .array results => All₂ (materializesAt depth codecs) values results
    | .object [] extra, .object results =>
      ∃ unsorted, All₂ (fun entry result => entry.1 = result.1 ∧
        materializesAt depth codecs entry.2 result.2) extra unsorted ∧
        (unsorted.map Prod.fst).Nodup ∧ results = orderedMembers unsorted
    | .map entries, .object results =>
      ∃ unsorted, All₂ (fun entry result => KeySpelling codecs.numbers entry.1 result.1 ∧
        materializesAt depth codecs entry.2 result.2) entries unsorted ∧
        (unsorted.map Prod.fst).Nodup ∧ results = orderedMembers unsorted
    | _, _ => False

def Materializes (codecs : ScalarCodecs) (value : Value) (wire : Wire) : Prop :=
  ∃ depth, materializesAt depth codecs value wire

def jsonValid (numbers : NumericCodec) (wire : Wire) : Bool :=
  match wire with
  | .null | .boolean _ | .text _ => true
  | .number text => (numbers.schemaNumber text).isSome
  | .array items => items.attach.all (fun item => jsonValid numbers item.val)
  | .object entries => decide (entries.map Prod.fst).Nodup &&
      entries.attach.all (fun entry => jsonValid numbers entry.val.2)
  | _ => false
termination_by sizeOf wire
decreasing_by
  · have smaller := List.sizeOf_lt_of_mem item.property
    simp only [Wire.array.sizeOf_spec]
    omega
  · rcases entry with ⟨⟨name, child⟩, member⟩
    have smaller := List.sizeOf_lt_of_mem member
    simp only [Wire.object.sizeOf_spec]
    simp only [Prod.mk.sizeOf_spec] at smaller
    omega

/-- Executable finite built-in Any materialization. Every key is checked before
constructing a snapshot. Source inputs cannot forge a snapshot or union evidence. -/
def materialize (codecs : ScalarCodecs) (value : Value) : Except Failure Wire :=
  match value with
  | .host _ payload => materialize codecs payload
  | .null => .ok .null
  | .nilBytes => .ok (.text (codecs.bytes.encode []))
  | .nilArray => .ok (.array [])
  | .nilMap => .ok (.object [])
  | .scalar scalar =>
    let wire := encodeScalar codecs .json scalar
    if jsonValid codecs.numbers wire then .ok wire else .error .invalid
  | .array values => do
    let results ← values.attach.mapM (fun value => materialize codecs value.val)
    return .array results
  | .object [] extra => do
    if !(decide (extra.map Prod.fst).Nodup) then .error .invalid
    else
      let results ← extra.attach.mapM fun entry => do
        let value ← materialize codecs entry.val.2
        return (entry.val.1, value)
      return .object (orderedMembers results)
  | .map entries => do
    let names ← nameKeys codecs.numbers (entries.map Prod.fst)
    let values ← entries.attach.mapM (fun entry => materialize codecs entry.val.2)
    return .object (orderedMembers (names.zip values))
  | _ => .error .unsupported
termination_by sizeOf value
decreasing_by
  · simp only [Value.host.sizeOf_spec]
    omega
  · have smaller := List.sizeOf_lt_of_mem value.property
    simp only [Value.array.sizeOf_spec]
    omega
  · rcases entry with ⟨⟨name, child⟩, member⟩
    have smaller := List.sizeOf_lt_of_mem member
    simp only [Value.object.sizeOf_spec, List.nil.sizeOf_spec]
    simp only [Prod.mk.sizeOf_spec] at smaller
    omega
  · rcases entry with ⟨⟨key, child⟩, member⟩
    have smaller := List.sizeOf_lt_of_mem member
    simp only [Value.map.sizeOf_spec]
    simp only [Prod.mk.sizeOf_spec] at smaller
    omega

end ValueContract.Candidate
