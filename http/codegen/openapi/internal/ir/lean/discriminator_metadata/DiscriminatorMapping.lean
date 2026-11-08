/-
Discriminator mapping regeneration and version projection (Loom issue #1005).

Mapping values are schema names or URI references (OAS 3.2 §4.25.3). Names
resolve against the entry document's authored component names (Appendix G.3);
URI references resolve through the document resolver (issue #957) to a
canonical target. A value that is not an explicit key is an implicit schema
name. The import records each explicit entry, in source order, as a key and
its resolved branch target; the exporter emits those entries as canonical
local references and never derives keys from Loom branch names.
-/
namespace Loom.OpenAPI.DiscriminatorMapping

abbrev Target := Nat
abbrev Key := String

inductive Ref where
  | name (n : String)
  | uri (u : String)
  deriving DecidableEq

def resolve (components : List (String × Target)) (resolver : String → Option Target) :
    Ref → Option Target
  | .name n => components.lookup n
  | .uri u => resolver u

/-- Effective mapping: an explicit entry wins, otherwise the value is an implicit
component name. -/
def effective (components : List (String × Target)) (explicit : List (Key × Ref))
    (resolver : String → Option Target) (k : Key) : Option Target :=
  match explicit.lookup k with
  | some r => resolve components resolver r
  | none => components.lookup k

/-- Import admission: every explicit value resolves to a listed branch. -/
def admitted (components : List (String × Target)) (explicit : List (Key × Ref))
    (resolver : String → Option Target) (branches : List Target) : Bool :=
  explicit.all fun e =>
    match resolve components resolver e.2 with
    | some t => branches.contains t
    | none => false

/-- Exporter emission from retained metadata: canonical local references. -/
def regenerate (components : List (String × Target)) (resolver : String → Option Target)
    (canonical : Target → String) (explicit : List (Key × Ref)) : List (Key × Ref) :=
  explicit.map fun e => (e.1, .uri (canonical ((resolve components resolver e.2).getD 0)))

theorem lookup_map_snd (f : Ref → Ref) (k : Key) :
    ∀ l : List (Key × Ref), (l.map fun e => (e.1, f e.2)).lookup k = (l.lookup k).map f
  | [] => rfl
  | (k', r) :: l => by
    simp only [List.map, List.lookup]
    split <;> simp_all [lookup_map_snd f k l]

theorem lookup_mem_all {p : Key × Ref → Bool} (k : Key) (r : Ref) :
    ∀ l : List (Key × Ref), l.all p = true → l.lookup k = some r → p (k, r) = true
  | [] => by simp [List.lookup]
  | (k', r') :: l => by
    intro hall hlook
    simp only [List.all, Bool.and_eq_true] at hall
    simp only [List.lookup] at hlook
    split at hlook
    · rename_i heq
      have hk : k = k' := by simpa using heq
      cases hlook
      subst hk
      exact hall.1
    · exact lookup_mem_all k r l hall.2 hlook

theorem lookup_append_none {α : Type} (k : Key) (l₁ l₂ : List (Key × α))
    (h : l₂.lookup k = none) : (l₁ ++ l₂).lookup k = l₁.lookup k := by
  induction l₁ with
  | nil => simpa using h
  | cons e l ih =>
    simp only [List.cons_append, List.lookup]
    split <;> simp_all

/-- Regeneration preserves the effective mapping of every key that is not the
name of a component added by import (promotion or bundling), provided the
regenerated document resolves each canonical reference to its target. -/
theorem regenerated_effective_mapping
    (components added : List (String × Target)) (explicit : List (Key × Ref))
    (resolver resolver' : String → Option Target) (canonical : Target → String)
    (branches : List Target)
    (hAdmit : admitted components explicit resolver branches = true)
    (hCanonical : ∀ t, resolver' (canonical t) = some t)
    (k : Key) (hAdded : added.lookup k = none) :
    effective (components ++ added) (regenerate components resolver canonical explicit) resolver' k =
      effective components explicit resolver k := by
  unfold effective regenerate
  rw [lookup_map_snd (fun r => Ref.uri (canonical ((resolve components resolver r).getD 0)))]
  cases hlook : explicit.lookup k with
  | none => simp [lookup_append_none k components added hAdded]
  | some r =>
    have hp := lookup_mem_all k r explicit hAdmit hlook
    simp only at hp
    cases hr : resolve components resolver r with
    | none => simp [hr] at hp
    | some t =>
      show resolve (components ++ added) resolver'
          (Ref.uri (canonical ((resolve components resolver r).getD 0))) = resolve components resolver r
      rw [hr]
      simp [resolve, hCanonical]

/-! Counterexamples to rejected mapping rules. -/

/-- Rejected rule from the earlier draft: a branch receives an implicit key only
when no explicit entry targets it. -/
def priorEffective (components : List (String × Target)) (explicit : List (Key × Ref))
    (resolver : String → Option Target) (k : Key) : Option Target :=
  match explicit.lookup k with
  | some r => resolve components resolver r
  | none =>
    match components.lookup k with
    | some t => if explicit.any (fun e => resolve components resolver e.2 == some t) then none else some t
    | none => none

theorem prior_implicit_rule_counterexample :
    effective [("Cat", 1)] [("cat", .name "Cat")] (fun _ => none) "Cat" = some 1 ∧
    priorEffective [("Cat", 1)] [("cat", .name "Cat")] (fun _ => none) "Cat" = none := by
  decide

/-- Rejected exporter rule: derive explicit keys from Loom branch names. Branch
2 is a promoted inline branch named `variant` that has no source key. -/
def deriveFromBranchNames (names : List (Key × Target)) (canonical : Target → String) :
    List (Key × Ref) :=
  names.map fun e => (e.1, .uri (canonical e.2))

theorem derived_keys_counterexample :
    effective [("Cat", 1)] [] (fun _ => none) "variant" = none ∧
    effective [("Cat", 1)]
      (deriveFromBranchNames [("Cat", 1), ("variant", 2)] (fun t => toString t))
      (fun u => if u = "2" then some 2 else if u = "1" then some 1 else none) "variant" = some 2 := by
  decide

/-- The hypothesis on added names is necessary: a promoted component name
becomes an implicit hint in the regenerated document. -/
theorem added_component_name_becomes_implicit_hint :
    effective [("Cat", 1)] [] (fun _ => none) "PetVariant" = none ∧
    effective ([("Cat", 1)] ++ [("PetVariant", 2)]) [] (fun _ => none) "PetVariant" = some 2 := by
  decide

/-! Version projection. A payload is abstracted as whether it carries the
discriminating property and whether its remaining members are valid. -/

abbrev Payload := Bool × Bool

structure Branch where
  requiresProperty : Bool
  accepts : Payload → Bool

def count (p : Payload) : List Branch → Nat
  | [] => 0
  | b :: bs => (if b.accepts p then 1 else 0) + count p bs

def unionAccepts (bs : List Branch) (p : Payload) : Bool := count p bs == 1

/-- Optional is derived from branch requiredness; it is never authored twice. -/
def optional (bs : List Branch) : Bool := bs.any fun b => !b.requiresProperty

inductive Version where
  | v30
  | v31
  | v32
  deriving DecidableEq

/-- Importer source versions (export targets are only 3.2.0 and 3.1.1).
OAS 3.2 §4.25.4: an optional property MUST have a defaultMapping. OAS 3.1.1
§4.8.25: the property SHOULD be required. OAS 3.0.3 §4.7.24.2.1: it MUST be
required. Only 3.2 has a defaultMapping field. -/
def admitsOptional (v : Version) (isOptional hasDefault : Bool) : Bool :=
  match v with
  | .v32 => !isOptional || hasDefault
  | .v31 => !hasDefault
  | .v30 => !hasDefault && !isOptional

def admitsVersion (v : Version) (bs : List Branch) (hasDefault : Bool) : Bool :=
  admitsOptional v (optional bs) hasDefault

/-- The existing 3.1 filter adds the property to the union's `required`. -/
def withUnionRequired (bs : List Branch) (p : Payload) : Bool := p.1 && unionAccepts bs p

theorem count_pos_exists (p : Payload) :
    ∀ bs : List Branch, 0 < count p bs → ∃ b ∈ bs, b.accepts p = true
  | [] => by simp [count]
  | b :: bs => by
    intro h
    by_cases hb : b.accepts p = true
    · exact ⟨b, by simp, hb⟩
    · simp [count, hb] at h
      obtain ⟨c, hc, hca⟩ := count_pos_exists p bs h
      exact ⟨c, by simp [hc], hca⟩

/-- For tagged envelopes every branch requires the property, so the 3.1
projection leaves validation unchanged. -/
theorem union_required_is_redundant_for_required_branches (bs : List Branch)
    (hSound : ∀ b ∈ bs, ∀ p, b.requiresProperty = true → b.accepts p = true → p.1 = true)
    (hRequired : ∀ b ∈ bs, b.requiresProperty = true) (p : Payload) :
    withUnionRequired bs p = unionAccepts bs p := by
  unfold withUnionRequired
  cases hu : unionAccepts bs p with
  | false => simp
  | true =>
    have hc : count p bs = 1 := by simpa [unionAccepts] using hu
    obtain ⟨b, hb, hacc⟩ := count_pos_exists p bs (by omega)
    have := hSound b hb p (hRequired b hb) hacc
    simp [this]

def catRequired : Branch := ⟨true, fun p => p.1 && p.2⟩
def otherPet : Branch := ⟨false, fun p => !p.1 && p.2⟩

/-- Applying that filter to an untagged union with a hint narrows validation:
the defaultMapping branch accepted a payload without the property. -/
theorem union_required_narrows_untagged_hint :
    unionAccepts [catRequired, otherPet] (false, true) = true ∧
    withUnionRequired [catRequired, otherPet] (false, true) = false := by decide

theorem version_rules_by_source_version :
    admitsVersion .v32 [catRequired, otherPet] false = false ∧
    admitsVersion .v31 [catRequired, otherPet] false = true ∧
    admitsVersion .v31 [catRequired] true = false ∧
    admitsVersion .v32 [catRequired, otherPet] true = true ∧
    admitsVersion .v30 [catRequired, otherPet] false = false ∧
    admitsVersion .v30 [catRequired] false = true := by decide

/-! Explicit keys, generated branch names and duplicate alternatives. -/

/-- `oneOf: [Cat, Dog]` with `mapping: {Cat: Dog}`: the explicit key shadows the
implicit `Cat`, both `Cat` and `Dog` hint to `Dog`, and `Cat` has no key. -/
theorem explicit_key_shadows_implicit :
    admitted [("Cat", 1), ("Dog", 2)] [("Cat", .name "Dog")] (fun _ => none) [1, 2] = true ∧
    effective [("Cat", 1), ("Dog", 2)] [("Cat", .name "Dog")] (fun _ => none) "Cat" = some 2 ∧
    effective [("Cat", 1), ("Dog", 2)] [("Cat", .name "Dog")] (fun _ => none) "Dog" = some 2 ∧
    effective ([("Cat", 1), ("Dog", 2)])
      (regenerate [("Cat", 1), ("Dog", 2)] (fun _ => none) (fun t => toString t) [("Cat", .name "Dog")])
      (fun u => if u = "1" then some 1 else if u = "2" then some 2 else none) "Cat" = some 2 := by
  decide

/-- Rejected validator: compare explicit keys with generated branch names (the
tagged-envelope rule). It refuses the valid document above, whose fallback
branch name for component `Cat` equals the explicit key `Cat`. -/
def rejectsByBranchNames (names : List (Key × Target)) (components : List (String × Target))
    (explicit : List (Key × Ref)) (resolver : String → Option Target) : Bool :=
  explicit.any fun e => names.any fun n =>
    n.1 == e.1 && some n.2 != resolve components resolver e.2

theorem branch_name_validation_counterexample :
    admitted [("Cat", 1), ("Dog", 2)] [("Cat", .name "Dog")] (fun _ => none) [1, 2] = true ∧
    rejectsByBranchNames [("Cat", 1), ("Dog", 2)] [("Cat", 1), ("Dog", 2)]
      [("Cat", .name "Dog")] (fun _ => none) = true := by
  decide

/-- Duplicate alternatives `oneOf: [A, A, B]`: admission and the mapping are
per target, so one key for `A` is emitted once and keeps its meaning. -/
theorem duplicate_alternatives_mapping :
    admitted [("A", 1), ("B", 2)] [("a", .name "A")] (fun _ => none) [1, 1, 2] = true ∧
    effective [("A", 1), ("B", 2)]
      (regenerate [("A", 1), ("B", 2)] (fun _ => none) (fun t => toString t) [("a", .name "A")])
      (fun u => if u = "1" then some 1 else if u = "2" then some 2 else none) "a" = some 1 ∧
    (regenerate [("A", 1), ("B", 2)] (fun _ => none) (fun t => toString t) [("a", .name "A")]).length = 1 := by
  decide

/-! The allOf parent form (ParentHint). Validation uses the parent schema; a
child's effective schema is the conjunction of its allOf members. -/

structure Child where
  inlineRequires : Bool
  otherRequires : Bool

/-- A child requires the property when any allOf member does, the parent included. -/
def childRequires (parentRequires : Bool) (c : Child) : Bool :=
  parentRequires || c.inlineRequires || c.otherRequires

/-- Optional is a fact of the parent, whose schema validates the payload. -/
def parentOptional (parentRequires : Bool) : Bool := !parentRequires

theorem inherited_requiredness (c : Child) : childRequires true c = true := by
  simp [childRequires]

/-- Rejected rule: derive optional from each child's inline schema only. The
typical `Cat: allOf[$ref Pet, {properties: …}]` inherits `Pet.required`, so it
is not optional; the inline-only rule would block it in 3.0 and demand a
defaultMapping in 3.2. -/
def inlineOnlyOptional (children : List Child) : Bool := children.any fun c => !c.inlineRequires

theorem inline_only_requiredness_counterexample :
    parentOptional true = false ∧
    admitsOptional .v30 (parentOptional true) false = true ∧
    admitsOptional .v32 (parentOptional true) false = true ∧
    inlineOnlyOptional [⟨false, false⟩] = true ∧
    admitsOptional .v30 (inlineOnlyOptional [⟨false, false⟩]) false = false ∧
    admitsOptional .v32 (inlineOnlyOptional [⟨false, false⟩]) false = false := by
  decide

theorem parent_hint_version_rules :
    admitsOptional .v32 (parentOptional false) false = false ∧
    admitsOptional .v32 (parentOptional false) true = true ∧
    admitsOptional .v31 (parentOptional false) false = true ∧
    admitsOptional .v30 (parentOptional false) false = false := by
  decide

end Loom.OpenAPI.DiscriminatorMapping
