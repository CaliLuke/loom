import ValueContract.ScalarSemantics

namespace ValueContract.Candidate

/-- Schema equality follows JSON numerical meaning, not number-token spelling.
Native protobuf constructors remain separate. This relation is deliberately not
raw-source equality or declared enum nil-container normalization. -/
def wireEquivalentAt : Nat → NumericCodec → Wire → Wire → Prop
  | 0, _, _, _ => False
  | depth + 1, numbers, left, right => match left, right with
    | .null, .null => True
    | .boolean left, .boolean right => left = right
    | .text left, .text right => left = right
    | .number left, .number right =>
      ∃ l r, numbers.schemaNumber left = some l ∧ numbers.schemaNumber right = some r ∧
        scalarEqual (.decimal l.coefficient l.exponent) (.decimal r.coefficient r.exponent) = true
    | .integer left, .integer right => left = right
    | .decimal lc le, .decimal rc re => scalarEqual (.decimal lc le) (.decimal rc re) = true
    | .bytes left, .bytes right => left = right
    | .array left, .array right => All₂ (wireEquivalentAt depth numbers) left right
    | .object left, .object right => left.length = right.length ∧
      (left.map Prod.fst).Nodup ∧ (right.map Prod.fst).Nodup ∧
      ∀ entry ∈ left, ∃ other ∈ right,
        entry.1 = other.1 ∧ wireEquivalentAt depth numbers entry.2 other.2
    | .oneof left value, .oneof right other =>
      left = right ∧ wireEquivalentAt depth numbers value other
    | _, _ => False

def wireEqualAt : Nat → NumericCodec → Wire → Wire → Bool
  | 0, _, _, _ => false
  | depth + 1, numbers, left, right => match left, right with
    | .null, .null => true
    | .boolean left, .boolean right => left == right
    | .text left, .text right => left == right
    | .number left, .number right =>
      (numbers.schemaNumber left).any fun l => (numbers.schemaNumber right).any fun r =>
        scalarEqual (.decimal l.coefficient l.exponent) (.decimal r.coefficient r.exponent)
    | .integer left, .integer right => left == right
    | .decimal lc le, .decimal rc re => scalarEqual (.decimal lc le) (.decimal rc re)
    | .bytes left, .bytes right => left == right
    | .array left, .array right => left.length == right.length &&
        (left.zip right).all (fun entry => wireEqualAt depth numbers entry.1 entry.2)
    | .object left, .object right => left.length == right.length &&
      decide (left.map Prod.fst).Nodup && decide (right.map Prod.fst).Nodup &&
      left.all (fun entry => right.any (fun other =>
        entry.1 == other.1 && wireEqualAt depth numbers entry.2 other.2))
    | .oneof left value, .oneof right other =>
      left == right && wireEqualAt depth numbers value other
    | _, _ => false

theorem wireEqualAt_iff (depth : Nat) (numbers : NumericCodec) (left right : Wire) :
    wireEqualAt depth numbers left right = true ↔ wireEquivalentAt depth numbers left right := by
  induction depth generalizing left right with
  | zero => simp [wireEqualAt, wireEquivalentAt]
  | succ depth ih =>
    cases left <;> cases right <;>
      simp [wireEqualAt, wireEquivalentAt, All₂, List.all_eq_true, List.any_eq_true,
        Option.any_eq_true, ih, and_assoc]
/-- Structural size computed by executable folds; kernel sizeOf is used only
for termination, never as a runtime dependency on compiler-generated instances. -/
def wireHeight (wire : Wire) : Nat := match wire with
  | .array entries => 1 + entries.attach.foldl (fun n entry => max n (wireHeight entry.val)) 0
  | .object entries => 1 + entries.attach.foldl (fun n entry => max n (wireHeight entry.val.2)) 0
  | .oneof _ child => 1 + wireHeight child
  | _ => 1
termination_by sizeOf wire
decreasing_by
  · have smaller := List.sizeOf_lt_of_mem entry.property
    simp only [Wire.array.sizeOf_spec]
    omega
  · rcases entry with ⟨⟨name, child⟩, member⟩
    have smaller := List.sizeOf_lt_of_mem member
    simp only [Wire.object.sizeOf_spec]
    simp only [Prod.mk.sizeOf_spec] at smaller
    omega
  · simp only [Wire.oneof.sizeOf_spec]
    omega

/-- The depth is derived from both finite wire trees, never a configured cap. -/
def wireEqual (numbers : NumericCodec) (left right : Wire) : Bool :=
  wireEqualAt (wireHeight left + wireHeight right + 1) numbers left right

end ValueContract.Candidate
