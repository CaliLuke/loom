import ValueContract.ValueEquality
import ValueContract.WireEquality

namespace ValueContract.Candidate

/-- Runtime enum values live in target observation space. Snapshot members use
JSON numerical equality; source Any equality remains the separate raw contract.
The source-to-target enum plan mapping is an explicit correspondence obligation. -/
def targetEquivalentAt : Nat → NumericCodec → Value → Value → Prop
  | 0, _, _, _ => False
  | depth + 1, numbers, left, right => match left, right with
    | .jsonSnapshot l, .jsonSnapshot r =>
      wireEquivalentAt (wireHeight l + wireHeight r + 1) numbers l r
    | .array l, .array r => All₂ (targetEquivalentAt depth numbers) l r
    | .object l le, .object r re => l.length = r.length ∧
      (∀ entry ∈ l, ∃ other ∈ r, entry.1 = other.1 ∧ targetEquivalentAt depth numbers entry.2 other.2) ∧
      le.length = re.length ∧
      (∀ entry ∈ le, ∃ other ∈ re, entry.1 = other.1 ∧ targetEquivalentAt depth numbers entry.2 other.2)
    | .map l, .map r => l.length = r.length ∧
      ∀ entry ∈ l, ∃ other ∈ r, SameKey numbers entry.1 other.1 ∧
        targetEquivalentAt depth numbers entry.2 other.2
    | .union li lb lv, .union ri rb rv =>
      li = ri ∧ lb = rb ∧ targetEquivalentAt depth numbers lv rv
    | _, _ => enumEquivalentAt numbers (depth + 1) left right

def targetEqualAt : Nat → NumericCodec → Value → Value → Bool
  | 0, _, _, _ => false
  | depth + 1, numbers, left, right => match left, right with
    | .jsonSnapshot l, .jsonSnapshot r => wireEqual numbers l r
    | .array l, .array r => l.length == r.length &&
      (l.zip r).all (fun entry => targetEqualAt depth numbers entry.1 entry.2)
    | .object l le, .object r re => l.length == r.length &&
      l.all (fun entry => r.any (fun other => entry.1 == other.1 &&
        targetEqualAt depth numbers entry.2 other.2)) &&
      le.length == re.length && le.all (fun entry => re.any (fun other =>
        entry.1 == other.1 && targetEqualAt depth numbers entry.2 other.2))
    | .map l, .map r => l.length == r.length &&
      l.all (fun entry => r.any (fun other => keyEqual numbers entry.1 other.1 &&
        targetEqualAt depth numbers entry.2 other.2))
    | .union li lb lv, .union ri rb rv =>
      li == ri && lb == rb && targetEqualAt depth numbers lv rv
    | _, _ => valueEqualAt numbers (depth + 1) true left right

theorem targetEqualAt_iff (depth : Nat) (numbers : NumericCodec) (left right : Value) :
    targetEqualAt depth numbers left right = true ↔ targetEquivalentAt depth numbers left right := by
  induction depth generalizing left right with
  | zero => simp [targetEqualAt, targetEquivalentAt]
  | succ depth ih =>
    cases left <;> cases right <;>
      simp [targetEqualAt, targetEquivalentAt, wireEqual, wireEqualAt_iff,
        valueEqualAt_enum_iff, keyEqual_iff, All₂, List.all_eq_true, List.any_eq_true, ih, and_assoc]

def targetEnumAllowed (numbers : NumericCodec) (members : Option (List Value)) (value : Value) : Bool :=
  members.all (fun entries => entries.any (fun member =>
    targetEqualAt (valueDepth value + valueDepth member + 1) numbers value member))

def TargetEnumAllows (numbers : NumericCodec) (members : Option (List Value)) (value : Value) : Prop :=
  match members with
  | none => True
  | some entries => ∃ member ∈ entries,
    targetEquivalentAt (valueDepth value + valueDepth member + 1) numbers value member

theorem targetEnumAllowed_iff (numbers : NumericCodec) (members : Option (List Value)) (value : Value) :
    targetEnumAllowed numbers members value = true ↔ TargetEnumAllows numbers members value := by
  cases members <;> simp [targetEnumAllowed, TargetEnumAllows, List.any_eq_true, targetEqualAt_iff]

end ValueContract.Candidate
