/-
Native-byte source shape, before semantic branch selection. Only containers
whose element type is exactly Go's builtin uint8 use this evidence. Defined
uint8 elements remain ordinary recursive arrays, including custom boundaries.
Canonical values and target plans do not contain this source-only shape.
-/
import ValueContract.SourceAdmission

namespace ValueContract.Candidate

/-- Each item retains its host equality class and exact builtin uint8 value.
A nil slice cannot carry elements; arrays cannot be nil. Defined container
identity is independent of the exact native element type and JSON binary shape.
Raw container DeepEqual evidence remains on the surrounding Input.host node. -/
inductive NativeByteSequence where
  | slice (definedContainer : Bool) (items : List (Nat × UInt8))
  | array (definedContainer : Bool) (items : List (Nat × UInt8))
  | nilSlice (definedContainer : Bool)
  deriving DecidableEq, Repr

/-- Original element evidence for declared Array child resolution. Integration
wraps each child with its host class and a SourceScalar builtinUInt8 tag;
it must not replace this list with untyped synthetic numeric children. -/
def NativeByteSequence.items : NativeByteSequence → List (Nat × UInt8)
  | .slice _ items | .array _ items => items
  | .nilSlice _ => []

/-- Octets used after source dispatch by declared Bytes or raw Any. Both native
byte arrays and slices use builtin JSON binary encoding; admission to declared
Bytes is a separate question. -/
def NativeByteSequence.octets (sequence : NativeByteSequence) : List UInt8 :=
  sequence.items.map Prod.snd

def NativeByteSequence.isNil : NativeByteSequence → Bool
  | .nilSlice _ => true
  | _ => false

/-- Primitive.Bytes admits concrete []byte, including nil, but neither a
fixed array nor a defined slice container. All roles share this source rule. -/
def NativeByteSequence.bytesAdmission : NativeByteSequence → Bool
  | .slice false _ | .nilSlice false => true
  | _ => false

/-- Independent pointwise octet relation; host class numbers affect neither
encoding nor admission, but remain available in the original item list. -/
inductive NativeBytePayloads : List (Nat × UInt8) → List UInt8 → Prop where
  | nil : NativeBytePayloads [] []
  | cons (rest : NativeBytePayloads items bytes) :
      NativeBytePayloads ((host, byte) :: items) (byte :: bytes)

inductive NativeByteOctets : NativeByteSequence → List UInt8 → Prop where
  | slice (payloads : NativeBytePayloads items bytes) :
      NativeByteOctets (.slice defined items) bytes
  | array (payloads : NativeBytePayloads items bytes) :
      NativeByteOctets (.array defined items) bytes
  | nilSlice : NativeByteOctets (.nilSlice defined) []

inductive NativeBytesAdmitted : NativeByteSequence → Prop where
  | slice : NativeBytesAdmitted (.slice false items)
  | nilSlice : NativeBytesAdmitted (.nilSlice false)

inductive NativeBytesNil : NativeByteSequence → Prop where
  | nilSlice : NativeBytesNil (.nilSlice defined)

theorem nativeBytePayloads_iff (items : List (Nat × UInt8)) (bytes : List UInt8) :
    items.map Prod.snd = bytes ↔ NativeBytePayloads items bytes := by
  constructor
  · intro equal
    subst bytes
    induction items with
    | nil => exact .nil
    | cons item rest ih => exact .cons ih
  · intro related
    induction related with
    | nil => rfl
    | cons _ ih => simp [ih]

/-- Octet extraction is independent of container naming and slice/array shape. -/
theorem nativeByteOctets_iff (sequence : NativeByteSequence) (bytes : List UInt8) :
    sequence.octets = bytes ↔ NativeByteOctets sequence bytes := by
  constructor
  · intro equal
    cases sequence with
    | slice defined items => exact .slice ((nativeBytePayloads_iff items bytes).mp equal)
    | array defined items => exact .array ((nativeBytePayloads_iff items bytes).mp equal)
    | nilSlice defined => cases equal; exact .nilSlice
  · intro related
    cases related with
    | slice payloads => exact (nativeBytePayloads_iff _ _).mpr payloads
    | array payloads => exact (nativeBytePayloads_iff _ _).mpr payloads
    | nilSlice => rfl

theorem nativeBytesAdmission_iff (sequence : NativeByteSequence) :
    sequence.bytesAdmission = true ↔ NativeBytesAdmitted sequence := by
  constructor
  · intro accepted
    cases sequence with
    | slice defined items => cases defined <;> simp_all [NativeByteSequence.bytesAdmission]; exact .slice
    | array defined items => simp [NativeByteSequence.bytesAdmission] at accepted
    | nilSlice defined => cases defined <;> simp_all [NativeByteSequence.bytesAdmission]; exact .nilSlice
  · intro admitted
    cases admitted <;> rfl

theorem nativeBytesAdmission_false_iff (sequence : NativeByteSequence) :
    sequence.bytesAdmission = false ↔ ¬ NativeBytesAdmitted sequence := by
  rw [← nativeBytesAdmission_iff]
  cases sequence.bytesAdmission <;> simp

theorem nativeBytesNil_iff (sequence : NativeByteSequence) :
    sequence.isNil = true ↔ NativeBytesNil sequence := by
  constructor
  · intro nil
    cases sequence <;> simp_all [NativeByteSequence.isNil]
    exact .nilSlice
  · intro nil
    cases nil
    rfl

theorem nativeBytesNil_empty {sequence : NativeByteSequence}
    (nil : sequence.isNil = true) : sequence.items = [] ∧ sequence.octets = [] := by
  cases (nativeBytesNil_iff sequence).mp nil
  exact ⟨rfl, rfl⟩

theorem nativeBytes_length (sequence : NativeByteSequence) :
    sequence.octets.length = sequence.items.length := by
  simp [NativeByteSequence.octets]

/-- Host evidence is preserved for array child dispatch, even though octet
extraction intentionally ignores it for builtin binary materialization. -/
theorem nativeBytes_children_preserved (defined : Bool) (items : List (Nat × UInt8)) :
    (NativeByteSequence.slice defined items).items = items ∧
    (NativeByteSequence.array defined items).items = items := ⟨rfl, rfl⟩

theorem nativeBytes_array_never_admitted (defined : Bool) (items : List (Nat × UInt8)) :
    (NativeByteSequence.array defined items).bytesAdmission = false := rfl

theorem nativeBytes_defined_slice_never_admitted (items : List (Nat × UInt8)) :
    (NativeByteSequence.slice true items).bytesAdmission = false ∧
    (NativeByteSequence.nilSlice true).bytesAdmission = false := ⟨rfl, rfl⟩

/-- Slice, fixed-array and named-container distinctions do not alter raw JSON
binary octets. They still alter declared Bytes source eligibility. -/
theorem nativeBytes_shape_control :
    (NativeByteSequence.slice false [(7, 104), (9, 105)]).bytesAdmission = true ∧
    (NativeByteSequence.slice true [(7, 104), (9, 105)]).bytesAdmission = false ∧
    (NativeByteSequence.array false [(7, 104), (9, 105)]).bytesAdmission = false ∧
    (NativeByteSequence.array true [(7, 104), (9, 105)]).octets = [104, 105] := by decide

/-- Nil and empty remain distinct presence states with equal empty octets. -/
theorem nativeBytes_nil_empty_control :
    (NativeByteSequence.nilSlice false).isNil = true ∧
    (NativeByteSequence.slice false []).isNil = false ∧
    (NativeByteSequence.array false []).isNil = false ∧
    (NativeByteSequence.nilSlice false).octets = [] ∧
    (NativeByteSequence.slice false []).octets = [] ∧
    (NativeByteSequence.array false []).octets = [] := by decide

end ValueContract.Candidate
