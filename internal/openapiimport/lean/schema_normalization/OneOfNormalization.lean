/-
Null admission in untagged `oneOf` normalization (Loom issue #1014).

A source `oneOf` has ordinary branches and `bareNulls` copies of the bare
`{type: null}` schema. Each ordinary branch accepts some non-null values and
admits null in one of three ways: not at all, at this occurrence (an inline
`type: [T, null]`), or through a referenced component that is declared
nullable. JSON Schema `oneOf` accepts a value when exactly one branch does,
and null is counted like every other value.

The Loom representation keeps ordinary branches (each still counting null
when it admits it) plus an occurrence-level `nullable` flag that accepts null
before branch matching. Bare null branches have no Loom carrier.

Branch null kinds are inputs here. `Any` admits null intrinsically and is
`declared`; a nested union's kind is derived from its own normalized
alternatives. That classification is checked separately, for bounded nested
trees, by `internal/openapiimport/tla/oneof_null/OneOfNull.tla`.
-/
namespace OpenAPIImport.OneOfNormalization

inductive NullKind where
  | excluded
  | occurrence
  | declared
  deriving DecidableEq, Repr

structure Branch (V : Type) where
  accepts : V → Bool
  null : NullKind

def admitsNull : NullKind → Bool
  | .excluded => false
  | _ => true

/-- `none` is JSON null; `some v` is a non-null value. -/
def Branch.matches {V : Type} (b : Branch V) : Option V → Bool
  | none => admitsNull b.null
  | some v => b.accepts v

def count {V : Type} (x : Option V) : List (Branch V) → Nat
  | [] => 0
  | b :: bs => (if b.matches x then 1 else 0) + count x bs

structure Source (V : Type) where
  branches : List (Branch V)
  bareNulls : Nat

def Source.accepts {V : Type} (s : Source V) : Option V → Bool
  | none => count none s.branches + s.bareNulls == 1
  | some v => count (some v) s.branches == 1

structure Rep (V : Type) where
  branches : List (Branch V)
  nullable : Bool

def Rep.accepts {V : Type} (r : Rep V) : Option V → Bool
  | none => r.nullable || count none r.branches == 1
  | some v => count (some v) r.branches == 1

def occurrences {V : Type} : List (Branch V) → Nat
  | [] => 0
  | b :: bs => (if b.null = .occurrence then 1 else 0) + occurrences bs

def strip {V : Type} (b : Branch V) : Branch V :=
  if b.null = .occurrence then { b with null := .excluded } else b

/--
Candidate normalization. Branch null admission is kept as authored and is
counted by exact-one matching. Bare null branches are removed:

* one bare null and no null-admitting branch: occurrence `nullable`;
* null already rejected by two or more counted alternatives: drop bare nulls
  when the remaining branches still reject null;
* one null-admitting branch beside a bare null: remove occurrence-local null,
  so null stays rejected; a declared nullable component cannot be changed
  here and is rejected (it needs a null-only branch, issue #1010);
* no ordinary branch: null-only domain, rejected (issue #1010).
-/
def normalize {V : Type} (s : Source V) : Option (Rep V) :=
  if s.branches = [] then none
  else if s.bareNulls = 0 then some ⟨s.branches, false⟩
  else if count none s.branches = 0 then some ⟨s.branches, s.bareNulls = 1⟩
  else if 2 ≤ count none s.branches then some ⟨s.branches, false⟩
  else if 0 < occurrences s.branches then some ⟨s.branches.map strip, false⟩
  else none

theorem count_some_strip {V : Type} (v : V) (bs : List (Branch V)) :
    count (some v) (bs.map strip) = count (some v) bs := by
  induction bs with
  | nil => rfl
  | cons b bs ih =>
    simp only [List.map, count, ih]
    congr 1
    unfold strip
    split <;> rfl

theorem count_none_strip {V : Type} (bs : List (Branch V)) :
    count none (bs.map strip) + occurrences bs = count none bs := by
  induction bs with
  | nil => rfl
  | cons b bs ih =>
    simp only [List.map, count, occurrences]
    cases h : b.null <;>
      simp [strip, h, Branch.matches, admitsNull] <;> omega

theorem normalize_sound {V : Type} (s : Source V) (r : Rep V)
    (h : normalize s = some r) (x : Option V) : r.accepts x = s.accepts x := by
  unfold normalize at h
  by_cases hEmpty : s.branches = []
  · simp [hEmpty] at h
  by_cases hBare : s.bareNulls = 0
  · simp [hEmpty, hBare] at h
    subst h
    cases x <;> simp [Rep.accepts, Source.accepts, hBare]
  by_cases hZero : count none s.branches = 0
  · simp [hEmpty, hBare, hZero] at h
    subst h
    cases x with
    | none =>
      by_cases hOne : s.bareNulls = 1 <;> simp [Rep.accepts, Source.accepts, hZero, hOne]
    | some v => simp [Rep.accepts, Source.accepts]
  by_cases hTwo : 2 ≤ count none s.branches
  · simp [hEmpty, hBare, hZero, hTwo] at h
    subst h
    cases x with
    | none =>
      have hRep : count none s.branches ≠ 1 := by omega
      have hSrc : count none s.branches + s.bareNulls ≠ 1 := by omega
      simp only [Rep.accepts, Source.accepts, Bool.false_or]
      rw [beq_false_of_ne hRep, beq_false_of_ne hSrc]
    | some v => simp [Rep.accepts, Source.accepts]
  by_cases hOcc : 0 < occurrences s.branches
  · simp [hEmpty, hBare, hZero, hTwo, hOcc] at h
    subst h
    have hStrip := count_none_strip s.branches
    cases x with
    | none =>
      have hRep : count none (s.branches.map strip) ≠ 1 := by omega
      have hSrc : count none s.branches + s.bareNulls ≠ 1 := by omega
      simp only [Rep.accepts, Source.accepts, Bool.false_or]
      rw [beq_false_of_ne hRep, beq_false_of_ne hSrc]
    | some v => simp [Rep.accepts, Source.accepts, count_some_strip]
  · simp [hEmpty, hBare, hZero, hTwo, hOcc] at h

/-! Concrete witnesses. `true` is a string value and `false` an integer. -/

def strNullHere : Branch Bool := ⟨fun v => v, .occurrence⟩
def str : Branch Bool := ⟨fun v => v, .excluded⟩
def strDeclaredNull : Branch Bool := ⟨fun v => v, .declared⟩
def int : Branch Bool := ⟨fun v => !v, .excluded⟩

/-- The previously reviewed collapse: strip branch nulls and make the
occurrence nullable whenever a bare null branch exists. -/
def naiveCollapse {V : Type} (s : Source V) : Rep V :=
  ⟨s.branches.map strip, decide (1 ≤ s.bareNulls)⟩

/-- Rejected candidate: ignore bare null branches. -/
def dropBare {V : Type} (s : Source V) : Rep V := ⟨s.branches, false⟩

theorem naive_collapse_counterexample :
    (naiveCollapse ⟨[strNullHere, int], 1⟩).accepts none = true ∧
    (Source.accepts ⟨[strNullHere, int], 1⟩ none) = false := by decide

theorem drop_bare_counterexample :
    (dropBare ⟨[str, int], 1⟩).accepts none = false ∧
    (Source.accepts ⟨[str, int], 1⟩ none) = true := by decide

/-- Observable shape of a representation: occurrence flag and branch null kinds. -/
def shape {V : Type} (r : Option (Rep V)) : Option (Bool × List NullKind) :=
  r.map fun r => (r.nullable, r.branches.map Branch.null)

theorem single_bare_null_becomes_occurrence_nullable :
    shape (normalize ⟨[str, int], 1⟩) = some (true, [.excluded, .excluded]) := by decide

theorem nullable_branch_counts_null :
    shape (normalize ⟨[strDeclaredNull, int], 0⟩) = some (false, [.declared, .excluded]) ∧
    (Rep.accepts ⟨[strDeclaredNull, int], false⟩ none) = true := by decide

theorem occurrence_null_is_removed_when_bare_null_rejects_null :
    shape (normalize ⟨[strNullHere, int], 1⟩) = some (false, [.excluded, .excluded]) := by decide

theorem two_nullable_branches_keep_null_rejected :
    shape (normalize ⟨[strNullHere, ⟨fun v => !v, .declared⟩], 1⟩) =
      some (false, [.occurrence, .declared]) := by decide

theorem declared_null_beside_bare_null_is_rejected :
    (normalize ⟨[strDeclaredNull, int], 1⟩).isNone = true := by decide

/-- `{}` (Any) admits null and cannot drop it: beside a bare null, null must
stay rejected, so the union is blocked like a declared nullable component. -/
def anyBranch : Branch Bool := ⟨fun _ => true, .declared⟩

theorem any_beside_bare_null_is_rejected :
    (normalize ⟨[anyBranch, str], 1⟩).isNone = true ∧
    (Source.accepts ⟨[anyBranch, str], 1⟩ none) = false := by decide

theorem any_without_bare_null_counts_null :
    shape (normalize ⟨[anyBranch, str], 0⟩) = some (false, [.declared, .excluded]) ∧
    (Rep.accepts ⟨[anyBranch, str], false⟩ none) = true := by decide

/-- Classifying `Any` as not admitting null (the rejected flag-only rule)
accepts the null the source rejects. -/
theorem any_as_excluded_counterexample :
    (Rep.accepts ⟨[{ anyBranch with null := .excluded }, str], true⟩ none) = true ∧
    (Source.accepts ⟨[anyBranch, str], 1⟩ none) = false := by decide

theorem null_only_domain_is_rejected :
    (normalize (⟨[], 1⟩ : Source Bool)).isNone = true := by decide

/-- Within the representation family available without a null-only branch
(keep the declared branch, choose the occurrence flag), no representation
rejects null as the source does. -/
theorem declared_null_beside_bare_null_has_no_kept_representation (n : Bool) :
    (Rep.accepts ⟨[strDeclaredNull, int], n⟩ none) ≠
      (Source.accepts ⟨[strDeclaredNull, int], 1⟩ none) := by
  cases n <;> decide

/-- Rejected alternative: adding null to another authored branch also gives
the source domain, but rewrites a branch schema the author kept non-null. -/
theorem null_injection_alternative_is_domain_equal (x : Option Bool) :
    (Rep.accepts ⟨[strDeclaredNull, { int with null := .occurrence }], false⟩ x) =
      (Source.accepts ⟨[strDeclaredNull, int], 1⟩ x) := by
  cases x with
  | none => decide
  | some v => cases v <;> decide

end OpenAPIImport.OneOfNormalization
