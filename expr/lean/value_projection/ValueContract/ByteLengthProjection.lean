import ValueContract.ScalarSemantics

namespace ValueContract.Candidate

/-- Number of encoded characters for complete base64 groups and one residue.
The final quantum has four characters for either nonzero residue. -/
def byteEncodedLength (groups residue : Nat) : Nat :=
  4 * groups + if residue = 0 then 0 else 4

/-- Independent decoded size of a residue branch. -/
def byteDecodedLength (groups residue : Nat) : Nat := 3 * groups + residue

/-- Integer group bounds are permitted to be negative internally. The emitter
omits empty branches and clamps the lower group bound at zero. -/
def byteGroupBounds (bounds : LengthBounds) (residue : Nat) : LengthBounds where
  minimum := bounds.minimum.map (fun lower => (lower - Int.ofNat residue + 2) / 3)
  maximum := bounds.maximum.map (fun upper => (upper - Int.ofNat residue) / 3)

/-- Encoded interval before omission of empty branches. A negative bound is
never emitted as a schema keyword: it identifies a branch without witnesses. -/
def byteEncodedBounds (bounds : LengthBounds) (residue : Nat) : LengthBounds :=
  let groups := byteGroupBounds bounds residue
  let tail : Int := if residue = 0 then 0 else 4
  { minimum := groups.minimum.map (fun lower => 4 * lower + tail)
    maximum := groups.maximum.map (fun upper => 4 * upper + tail) }

/-- Ceiling/floor residue bounds preserve every decoded length, including
negative authored bounds and inconsistent ranges. No fixed size bound is used. -/
theorem byteGroupBounds_iff (bounds : LengthBounds) (groups residue : Nat) :
    lengthAllowed (byteGroupBounds bounds residue) groups = true ↔
      lengthAllowed bounds (byteDecodedLength groups residue) = true := by
  simp only [lengthAllowed, Bool.and_eq_true, Option.all_eq_true, decide_eq_true_eq]
  rcases bounds with ⟨lower, upper⟩
  cases lower <;> cases upper <;>
    simp [byteGroupBounds, byteDecodedLength] <;> omega

/-- Each residue's encoded interval is exactly its decoded-length interval.
The grammar supplies the residue, so equal encoded sizes across residues cannot
collapse the one-, two-, and three-byte boundary distinctions. -/
theorem byteEncodedBounds_iff (bounds : LengthBounds) (groups residue : Nat) :
    lengthAllowed (byteEncodedBounds bounds residue) (byteEncodedLength groups residue) = true ↔
      lengthAllowed bounds (byteDecodedLength groups residue) = true := by
  rw [← byteGroupBounds_iff bounds groups residue]
  rcases bounds with ⟨lower, upper⟩
  cases lower <;> cases upper <;> by_cases zero : residue = 0 <;>
    simp [lengthAllowed, byteEncodedBounds, byteGroupBounds, byteEncodedLength, zero] <;> omega

/-- Actual branch emission clamps vacuous lower bounds and omits empty ranges.
Every retained keyword is nonnegative; integer representation is checked by Go. -/
def byteProjectedBranch (bounds : LengthBounds) (residue : Nat) : Option LengthBounds :=
  let groups := byteGroupBounds bounds residue
  let lower := max 0 (groups.minimum.getD 0)
  let tail : Int := if residue = 0 then 0 else 4
  if groups.maximum.any (fun upper => upper < lower) then none
  else some { minimum := some (4 * lower + tail)
              maximum := groups.maximum.map (fun upper => 4 * upper + tail) }

/-- Clamping and branch omission preserve the exact interval. Empty residue
branches have no witnesses; omitted branches never silently relax the range. -/
theorem byteProjectedBranch_iff (bounds : LengthBounds) (groups residue : Nat) :
    (byteProjectedBranch bounds residue).any
      (fun projected => lengthAllowed projected (byteEncodedLength groups residue)) = true ↔
      lengthAllowed bounds (byteDecodedLength groups residue) = true := by
  rw [← byteGroupBounds_iff bounds groups residue]
  unfold byteProjectedBranch
  generalize byteGroupBounds bounds residue = interval
  rcases interval with ⟨lower, upper⟩
  simp only [lengthAllowed, Bool.and_eq_true, Option.all_eq_true, decide_eq_true_eq]
  cases lower <;> cases upper <;> by_cases zero : residue = 0 <;>
    simp [byteEncodedLength, zero]
  all_goals try split
  all_goals try simp_all only [Option.any_none, Option.any_some,
    Option.all_some, Bool.and_eq_true, decide_eq_true_eq, Bool.false_eq_true]
  all_goals try simp only [Nat.cast_add, Nat.cast_mul, Nat.cast_ofNat, Nat.cast_zero]
  all_goals try simp only [false_iff]
  all_goals try intro impossible
  all_goals omega

/-- Every nonnegative decoded size belongs to exactly its ordinary division
residue. This covers arbitrarily large finite byte sequences. -/
theorem byteResidues_cover (length : Nat) :
    length % 3 < 3 ∧ byteDecodedLength (length / 3) (length % 3) = length := by
  simp only [byteDecodedLength]
  omega

/-- The final quantum counts the data alphabet characters and literal padding.
Its unrestricted alphabet does not require zero unused padding bits. -/
def ByteAlphabet (character : Char) : Prop :=
  ('A' ≤ character ∧ character ≤ 'Z') ∨ ('a' ≤ character ∧ character ≤ 'z') ∨
  ('0' ≤ character ∧ character ≤ '9') ∨ character = '+' ∨ character = '/'

/-- Independent base64 grammar over characters. Codec decoding and concrete
ECMA-262 regex execution remain separately tested boundaries. -/
def ByteGrammar (groups residue : Nat) (characters : List Char) : Prop :=
  residue < 3 ∧ ∃ data,
    (∀ character ∈ data, ByteAlphabet character) ∧
    data.length = 4 * groups + (if residue = 0 then 0 else residue + 1) ∧
    characters = data ++ List.replicate (if residue = 0 then 0 else 3 - residue) '='

theorem byteGrammar_length {groups residue : Nat} {characters : List Char}
    (grammar : ByteGrammar groups residue characters) :
    characters.length = byteEncodedLength groups residue := by
  obtain ⟨small, data, _, length, rfl⟩ := grammar
  simp only [List.length_append, List.length_replicate, length, byteEncodedLength]
  by_cases zero : residue = 0 <;> simp [zero] <;> omega

/-- Grammar membership plus projected string bounds has exactly the decoded
length predicate. This theorem makes no claim about schema enums or nulls. -/
theorem byteGrammar_bounds_iff {bounds : LengthBounds} {groups residue : Nat}
    {characters : List Char} (grammar : ByteGrammar groups residue characters) :
    lengthAllowed (byteEncodedBounds bounds residue) characters.length = true ↔
      lengthAllowed bounds (byteDecodedLength groups residue) = true := by
  rw [byteGrammar_length grammar]
  exact byteEncodedBounds_iff bounds groups residue


/-- The emitted, clamped branch has exactly the decoded bound predicate for
all strings in its grammar. Empty branches are represented by no schema. -/
theorem byteGrammar_projected_iff {bounds : LengthBounds} {groups residue : Nat}
    {characters : List Char} (grammar : ByteGrammar groups residue characters) :
    (byteProjectedBranch bounds residue).any
      (fun projected => lengthAllowed projected characters.length) = true ↔
      lengthAllowed bounds (byteDecodedLength groups residue) = true := by
  rw [byteGrammar_length grammar]
  exact byteProjectedBranch_iff bounds groups residue

/-- Checked multiplication can be decided before computing an encoded size.
This is an integer-representation check, not a bound on the semantic theorem. -/
theorem byteEncodedLength_limit (groups residue limit : Nat) :
    byteEncodedLength groups residue ≤ limit ↔
      (if residue = 0 then 0 else 4) ≤ limit ∧
      groups ≤ (limit - (if residue = 0 then 0 else 4)) / 4 := by
  unfold byteEncodedLength
  by_cases zero : residue = 0 <;> simp [zero] <;> omega

/-- Empty bytes are admitted by an unbounded grammar, including a nil byte
slice's JSON string representation. JSON null is outside this grammar theorem. -/
theorem byteEmpty_unbounded :
    (byteProjectedBranch {} 0).any (fun bounds => lengthAllowed bounds 0) = true := by
  decide

theorem byteNegativeUpper_empty :
    byteProjectedBranch { maximum := some (-1) } 0 = none ∧
    byteProjectedBranch { maximum := some (-1) } 1 = none ∧
    byteProjectedBranch { maximum := some (-1) } 2 = none := by
  decide

theorem byteMinimumTwo_excludesOne :
    (byteProjectedBranch { minimum := some 2 } 1).any
      (fun bounds => lengthAllowed bounds 4) = false := by
  decide

theorem byteMaximumTwo_acceptsTwo :
    (byteProjectedBranch { maximum := some 2 } 2).any
      (fun bounds => lengthAllowed bounds 4) = true := by
  decide

/-- Noncanonical unused pad bits remain within the grammar. The codec boundary
is tested separately; no canonical-bit condition is smuggled into the model. -/
theorem byteNoncanonicalPadBits : ByteGrammar 0 2 ['a', 'G', 'l', '='] := by
  refine ⟨by decide, ['a', 'G', 'l'], ?_, by decide, by decide⟩
  intro character member
  simp only [List.mem_cons, List.not_mem_nil, or_false] at member
  rcases member with rfl | rfl | rfl <;> unfold ByteAlphabet <;> decide

end ValueContract.Candidate
