import ValueContract.Typing
import ValueContract.ValueDepth

namespace ValueContract.Candidate

variable {keys : KeyCodec}

/-- Snapshot equality preserves exact JSON number spelling. This is stronger
than JSONValueEqual's rational-number equivalence and is used only for the
canonical materialize/decode preservation theorem, never schema enum matching.
Object order is irrelevant; duplicate keys are explicitly rejected. -/
def wireSameAt : Nat → Wire → Wire → Bool
  | 0, _, _ => false
  | depth + 1, left, right => match left, right with
    | .null, .null => true
    | .boolean l, .boolean r => l == r
    | .number l, .number r | .text l, .text r => l == r
    | .integer l, .integer r => l == r
    | .decimal lc le, .decimal rc re => lc == rc && le == re
    | .bytes l, .bytes r => l == r
    | .array l, .array r => l.length == r.length &&
        (l.zip r).all (fun pair => wireSameAt depth pair.1 pair.2)
    | .object l, .object r => decide (l.map Prod.fst).Nodup &&
        decide (r.map Prod.fst).Nodup && l.length == r.length &&
        l.all (fun entry => r.any (fun other =>
          entry.1 == other.1 && wireSameAt depth entry.2 other.2))
    | .oneof ln l, .oneof rn r => ln == rn && wireSameAt depth l r
    | _, _ => false

/-- Independent exact structured snapshot relation. -/
def WireSameAt : Nat → Wire → Wire → Prop
  | 0, _, _ => False
  | depth + 1, left, right => match left, right with
    | .null, .null => True
    | .boolean l, .boolean r => l = r
    | .number l, .number r | .text l, .text r => l = r
    | .integer l, .integer r => l = r
    | .decimal lc le, .decimal rc re => lc = rc ∧ le = re
    | .bytes l, .bytes r => l = r
    | .array l, .array r => All₂ (WireSameAt depth) l r
    | .object l, .object r => (l.map Prod.fst).Nodup ∧
        (r.map Prod.fst).Nodup ∧ l.length = r.length ∧
        ∀ entry ∈ l, ∃ other ∈ r,
          entry.1 = other.1 ∧ WireSameAt depth entry.2 other.2
    | .oneof ln l, .oneof rn r => ln = rn ∧ WireSameAt depth l r
    | _, _ => False

theorem wireSameAt_iff (depth : Nat) (left right : Wire) :
    wireSameAt depth left right = true ↔ WireSameAt depth left right := by
  induction depth generalizing left right with
  | zero => simp [wireSameAt, WireSameAt]
  | succ depth ih => cases left <;> cases right <;>
      simp [wireSameAt, WireSameAt, All₂, List.all_eq_true, List.any_eq_true, ih, and_assoc]

/-- Raw Any enum comparison follows its own legacy relation rather than
declared collection normalization or target JSON decoding. -/
def rawAnyEqualLayer (recursive : Value → Value → Bool) (left right : Value) : Bool :=
  match left, right with
  | .host identity payload, .host other otherPayload =>
    identity == other || recursive payload otherPayload
  | .host _ payload, right => recursive payload right
  | left, .host _ payload => recursive left payload
  | left, right =>
    if rawNil left || rawNil right then rawNil left && rawNil right
    else match left, right with
    | .scalar l, .scalar r => scalarEqual l r
    | _, _ => match rawObject left, rawObject right with
      | some l, some r => l.length == r.length &&
        l.all (fun entry => r.any (fun other =>
          entry.1 == other.1 && recursive entry.2 other.2))
      | _, _ => match rawSlice left, rawSlice right with
        | some l, some r => l.length == r.length &&
          (l.zip r).all (fun pair => recursive pair.1 pair.2)
        | _, _ => false

def rawAnyEqualAt : Nat → Value → Value → Bool
  | 0, _, _ => false
  | depth + 1, left, right => rawAnyEqualLayer (rawAnyEqualAt depth) left right

theorem rawAnyEqualLayer_iff (check : Value → Value → Bool) (relation : Value → Value → Prop)
    (correspondence : ∀ left right, check left right = true ↔ relation left right)
    (left right : Value) :
    rawAnyEqualLayer check left right = true ↔ rawAnyLayer relation left right := by
  cases left <;> cases right <;>
    simp [rawAnyEqualLayer, rawAnyLayer, rawNil, rawObject, rawSlice, All₂,
      List.all_eq_true, correspondence]
  all_goals split <;> simp_all [List.all_eq_true, List.any_eq_true]
  all_goals split <;> simp_all [All₂, List.all_eq_true, List.any_eq_true, and_assoc]

theorem rawAnyEqualAt_iff (depth : Nat) (left right : Value) :
    rawAnyEqualAt depth left right = true ↔ rawAnyEquivalentAt depth left right := by
  induction depth generalizing left right with
  | zero => simp [rawAnyEqualAt, rawAnyEquivalentAt]
  | succ depth ih => exact rawAnyEqualLayer_iff _ _ ih left right

theorem rawAnyEqualLayer_at_iff (depth : Nat) (left right : Value) :
    rawAnyEqualLayer (rawAnyEqualAt depth) left right = true ↔
      rawAnyLayer (rawAnyEquivalentAt depth) left right :=
  rawAnyEqualLayer_iff _ _ (rawAnyEqualAt_iff depth) left right

/-- Executable structural comparison. Only enum/default normalization enables
the nil-to-empty cases. Object/map order is irrelevant; array order and union
occurrence/branch identities remain observable. Typed inputs separately reject
duplicates, so the existential member lookup cannot collapse their multiplicity. -/
def valueEqualAt (keys : KeyCodec) : Nat → Bool → Value → Value → Bool
  | 0, _, _, _ => false
  | depth + 1, enumMode, left, right =>
    match left, right with
    | .host identity payload, right =>
      if enumMode then rawAnyEqualLayer (rawAnyEqualAt depth) (.host identity payload) right
      else match right with
        | .host _ otherPayload => valueEqualAt keys depth false payload otherPayload
        | _ => valueEqualAt keys depth false payload right
    | left, .host identity payload =>
      if enumMode then rawAnyEqualLayer (rawAnyEqualAt depth) left (.host identity payload)
      else valueEqualAt keys depth false left payload
    | .absent, .absent | .null, .null | .nilArray, .nilArray | .nilMap, .nilMap => true
    | .nilBytes, .nilBytes => !enumMode
    | .nilArray, .array [] | .array [], .nilArray => enumMode
    | .nilMap, .map [] | .map [], .nilMap => enumMode
    | .scalar l, .scalar r => (enumMode || l.kind == r.kind) && scalarEqual l r
    | .array l, .array r =>
      l.length == r.length && (l.zip r).all (fun pair =>
        valueEqualAt keys depth enumMode pair.1 pair.2)
    | .object l le, .object r re =>
      l.length == r.length &&
      l.all (fun entry => r.any (fun other =>
        entry.1 == other.1 && valueEqualAt keys depth enumMode entry.2 other.2)) &&
      le.length == re.length &&
      le.all (fun entry => re.any (fun other =>
        entry.1 == other.1 && valueEqualAt keys depth enumMode entry.2 other.2))
    | .map l, .map r => l.length == r.length &&
      l.all (fun entry => r.any (fun other =>
        keyEqual keys entry.1 other.1 && valueEqualAt keys depth enumMode entry.2 other.2))
    | .union identity branch payload, .union other otherBranch otherPayload =>
      identity == other && branch == otherBranch &&
        valueEqualAt keys depth enumMode payload otherPayload
    | .jsonSnapshot l, .jsonSnapshot r => !enumMode && wireSameAt depth l r
    | .any l, .any r => if enumMode then rawAnyEqualAt depth l r
      else valueEqualAt keys depth false l r
    | _, _ => false

/-- The Boolean enum comparison implements the pre-existing independent typing
relation at every derivation depth, rather than defining equality by execution. -/
theorem valueEqualAt_enum_iff (depth : Nat) (left right : Value) :
    valueEqualAt keys depth true left right = true ↔ enumEquivalentAt keys depth left right := by
  induction depth generalizing left right with
  | zero => simp [valueEqualAt, enumEquivalentAt]
  | succ depth ih =>
    cases left <;> cases right <;>
      simp [valueEqualAt, enumEquivalentAt, All₂, List.all_eq_true,
        List.any_eq_true, ih, keyEqual_iff, rawAnyEqualAt_iff, rawAnyEqualLayer_at_iff, and_assoc]
    all_goals split <;> simp_all

/-- Strict equality for semantic/observation use never introduces enum-only
nil-container normalization. It retains absent and explicit null separately. -/
def strictEquivalentAt (keys : KeyCodec) : Nat → Value → Value → Prop
  | 0, _, _ => False
  | depth + 1, left, right =>
    match left, right with
    | .host _ payload, .host _ otherPayload => strictEquivalentAt keys depth payload otherPayload
    | .host _ payload, right => strictEquivalentAt keys depth payload right
    | left, .host _ payload => strictEquivalentAt keys depth left payload
    | .absent, .absent | .null, .null | .nilArray, .nilArray | .nilMap, .nilMap
    | .nilBytes, .nilBytes => True
    | .scalar l, .scalar r => l.kind = r.kind ∧ scalarEqual l r = true
    | .array l, .array r => All₂ (strictEquivalentAt keys depth) l r
    | .object l le, .object r re =>
      l.length = r.length ∧
      (∀ entry ∈ l, ∃ other ∈ r, entry.1 = other.1 ∧
        strictEquivalentAt keys depth entry.2 other.2) ∧
      le.length = re.length ∧
      (∀ entry ∈ le, ∃ other ∈ re, entry.1 = other.1 ∧
        strictEquivalentAt keys depth entry.2 other.2)
    | .map l, .map r => l.length = r.length ∧
      ∀ entry ∈ l, ∃ other ∈ r,
        SameKey keys entry.1 other.1 ∧
        strictEquivalentAt keys depth entry.2 other.2
    | .union identity branch payload, .union other otherBranch otherPayload =>
      identity = other ∧ branch = otherBranch ∧
        strictEquivalentAt keys depth payload otherPayload
    | .jsonSnapshot l, .jsonSnapshot r => WireSameAt depth l r
    | .any l, .any r => strictEquivalentAt keys depth l r
    | _, _ => False

def StrictEquivalent (keys : KeyCodec) (left right : Value) : Prop :=
  ∃ depth, strictEquivalentAt keys depth left right

theorem valueEqualAt_strict_iff (depth : Nat) (left right : Value) :
    valueEqualAt keys depth false left right = true ↔ strictEquivalentAt keys depth left right := by
  induction depth generalizing left right with
  | zero => simp [valueEqualAt, strictEquivalentAt]
  | succ depth ih =>
    cases left <;> cases right <;>
      simp [valueEqualAt, strictEquivalentAt, All₂, List.all_eq_true,
        List.any_eq_true, ih, keyEqual_iff, wireSameAt_iff, and_assoc]
    all_goals split <;> simp_all

end ValueContract.Candidate
