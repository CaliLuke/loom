import ValueContract.Typing

namespace ValueContract.Candidate

/-- Executable lookup preserves the full declaration. Graph validation separately
establishes identity uniqueness, so traversal order cannot choose a different node. -/
def findDeclaration (declarations : Declarations) (identity : Identity) : Option Declaration :=
  declarations.find? (fun declaration => declaration.identity == identity)

/-- Local identity/name checks preserve supported cross-member source-name versus
wire-alias overlap. Only duplicates within each individual namespace are rejected. -/
def declarationNamesValid : Contract → Bool
  | .object members _ =>
      decide (members.map Member.identity).Nodup &&
      decide (members.map Member.sourceName).Nodup &&
      decide (members.map Member.wireAlias).Nodup
  | .union _ alternatives => decide (alternatives.map Alternative.identity).Nodup
  | _ => true

/-- Validate finite table structure, not semantic enum members or DSL extraction.
Ranks decrease only on same-input expansion edges; recursive object/collection
children need existence, not decreasing ranks or a fixed inhabitant-depth bound. -/
def validateDeclarations (declarations : Declarations) : Bool :=
  decide (declarations.map Declaration.identity).Nodup &&
  declarations.all (fun declaration =>
    (nonconsumingChildren declaration.contract).all (fun child =>
      declarations.any (fun target => target.identity == child &&
        decide (target.expansionRank < declaration.expansionRank))) &&
    (consumingChildren declaration.contract).all (fun child =>
      declarations.any (fun target => target.identity == child)) &&
    declarationNamesValid declaration.contract)

/-- The executable checker is both sound and complete for the independently
defined graph predicate. An always-reject checker cannot satisfy this equivalence. -/
theorem validateDeclarations_iff (declarations : Declarations) :
    validateDeclarations declarations = true ↔ WellFormedDeclarations declarations := by
  have names (contract : Contract) : declarationNamesValid contract = true ↔
      (match contract with
      | .object members _ =>
          (members.map Member.identity).Nodup ∧
          (members.map Member.sourceName).Nodup ∧
          (members.map Member.wireAlias).Nodup
      | .union _ alternatives => (alternatives.map Alternative.identity).Nodup
      | _ => True) := by
    cases contract <;> simp [declarationNamesValid, and_assoc]
  simp only [validateDeclarations, WellFormedDeclarations, Bool.and_eq_true,
    decide_eq_true_eq, List.all_eq_true, List.any_eq_true, beq_iff_eq, names, and_assoc]
  rfl

private theorem findByIdentity_of_mem {entries : List α} (key : α → Identity)
    (unique : (entries.map key).Nodup) {entry : α} (member : entry ∈ entries) :
    entries.find? (fun candidate => key candidate == key entry) = some entry := by
  induction entries with
  | nil => simp at member
  | cons head tail ih =>
    have separated := List.nodup_cons.mp (show (key head :: tail.map key).Nodup from unique)
    rcases List.mem_cons.mp member with equal | inside
    · subst entry
      simp
    · have different : key head ≠ key entry := by
        intro equal
        exact separated.1 (List.mem_map.mpr ⟨entry, inside, equal.symm⟩)
      simpa [List.find?, beq_eq_false_iff_ne.mpr different] using ih separated.2 inside

theorem findDeclaration_mem {declarations : Declarations} {identity : Identity}
    {declaration : Declaration} (found : findDeclaration declarations identity = some declaration) :
    declaration ∈ declarations :=
  List.mem_of_find?_eq_some found

theorem findDeclaration_identity {declarations : Declarations} {identity : Identity}
    {declaration : Declaration} (found : findDeclaration declarations identity = some declaration) :
    declaration.identity = identity := by
  exact beq_iff_eq.mp (List.find?_some
    (p := fun candidate : Declaration => candidate.identity == identity) found)

theorem findDeclaration_of_mem {declarations : Declarations}
    (wellFormed : WellFormedDeclarations declarations) {declaration : Declaration}
    (member : declaration ∈ declarations) :
    findDeclaration declarations declaration.identity = some declaration :=
  findByIdentity_of_mem Declaration.identity wellFormed.1 member

/-- A recursive resolver can obtain a strictly smaller same-input rank from
successful lookup; no existential depth witness or arbitrary fuel is assumed. -/
theorem declaration_nonconsuming_rank {declarations : Declarations}
    (wellFormed : WellFormedDeclarations declarations) {identity child : Identity}
    {declaration : Declaration} (found : findDeclaration declarations identity = some declaration)
    (edge : child ∈ nonconsumingChildren declaration.contract) :
    ∃ target, findDeclaration declarations child = some target ∧
      target.expansionRank < declaration.expansionRank := by
  obtain ⟨target, member, sameIdentity, smaller⟩ :=
    (wellFormed.2 declaration (findDeclaration_mem found)).1 child edge
  exact ⟨target, sameIdentity ▸ findDeclaration_of_mem wellFormed member, smaller⟩

/-- Consuming children exist even when they refer back to their parent or an
ancestor. Value descent, rather than a graph-rank decrease, owns that recursion. -/
theorem declaration_consuming_exists {declarations : Declarations}
    (wellFormed : WellFormedDeclarations declarations) {identity child : Identity}
    {declaration : Declaration} (found : findDeclaration declarations identity = some declaration)
    (edge : child ∈ consumingChildren declaration.contract) :
    ∃ target, findDeclaration declarations child = some target := by
  obtain ⟨target, member, sameIdentity⟩ :=
    (wellFormed.2 declaration (findDeclaration_mem found)).2.1 child edge
  exact ⟨target, sameIdentity ▸ findDeclaration_of_mem wellFormed member⟩

/-- Wire matching does not consume a wire node through a selector, alias,
nullable/non-null edge, or untagged union. Project/Observe may consume a semantic child
at a selector; this stronger rank is specifically the schema/decoder guard. -/
def targetNonconsumingChildren : Target → List Identity
  | .nullable child | .nonNull child | .alias child | .select _ child => [child]
  | .union _ .untagged alternatives => alternatives.map TargetAlternative.child
  | _ => []

/-- Objects, arrays, and maps consume wire children. Tagged and protobuf unions
consume an envelope payload; untagged alternatives do not have that envelope. -/
def targetConsumingChildren : Target → List Identity
  | .array child _ | .map _ _ child _ => [child]
  | .object members _ => members.map TargetMember.child
  | .union _ (.tagged _ _) alternatives | .union _ .protobuf alternatives =>
      alternatives.map TargetAlternative.child
  | _ => []

/-- Independent target graph well-formedness. Scalar codec/enum validity is a
separate semantic obligation. Cross-namespace authored aliases are not rejected.
Ranks constrain only same-wire expansion, not the depth of finite inhabitants. -/
def WellFormedTargets (targets : Targets) : Prop :=
  (targets.map TargetDeclaration.identity).Nodup ∧
  ∀ declaration ∈ targets,
    (∀ child ∈ targetNonconsumingChildren declaration.target,
      ∃ target ∈ targets, target.identity = child ∧
        target.expansionRank < declaration.expansionRank) ∧
    (∀ child ∈ targetConsumingChildren declaration.target,
      ∃ target ∈ targets, target.identity = child) ∧
    match declaration.target with
    | .object members _ =>
        (members.map TargetMember.identity).Nodup ∧
        (members.map TargetMember.wireName).Nodup
    | .union _ style alternatives =>
        (alternatives.map TargetAlternative.identity).Nodup ∧
        (alternatives.map TargetAlternative.wireName).Nodup ∧
        (match style with | .tagged tagKey valueKey => tagKey ≠ valueKey | _ => True)
    | _ => True

/-- Local target namespace checks; representation semantics remain separate. -/
def targetNamesValid : Target → Bool
  | .object members _ =>
      decide (members.map TargetMember.identity).Nodup &&
      decide (members.map TargetMember.wireName).Nodup
  | .union _ style alternatives =>
      decide (alternatives.map TargetAlternative.identity).Nodup &&
      decide (alternatives.map TargetAlternative.wireName).Nodup &&
      (match style with | .tagged tagKey valueKey => tagKey != valueKey | _ => true)
  | _ => true

/-- Check the finite target table without fuel or a fixed value-depth cutoff. -/
def validateTargets (targets : Targets) : Bool :=
  decide (targets.map TargetDeclaration.identity).Nodup &&
  targets.all (fun declaration =>
    (targetNonconsumingChildren declaration.target).all (fun child =>
      targets.any (fun target => target.identity == child &&
        decide (target.expansionRank < declaration.expansionRank))) &&
    (targetConsumingChildren declaration.target).all (fun child =>
      targets.any (fun target => target.identity == child)) &&
    targetNamesValid declaration.target)

/-- Soundness and completeness prevent both accepting malformed graphs and
silently rejecting supported recursive graphs or valid target namespace plans. -/
theorem validateTargets_iff (targets : Targets) :
    validateTargets targets = true ↔ WellFormedTargets targets := by
  have names (target : Target) : targetNamesValid target = true ↔
      (match target with
      | .object members _ =>
          (members.map TargetMember.identity).Nodup ∧
          (members.map TargetMember.wireName).Nodup
      | .union _ style alternatives =>
          (alternatives.map TargetAlternative.identity).Nodup ∧
          (alternatives.map TargetAlternative.wireName).Nodup ∧
          (match style with | .tagged tagKey valueKey => tagKey ≠ valueKey | _ => True)
      | _ => True) := by
    cases target <;> simp [targetNamesValid, and_assoc]
    case union occurrence style alternatives =>
      cases style <;> simp
  simp only [validateTargets, WellFormedTargets, Bool.and_eq_true,
    decide_eq_true_eq, List.all_eq_true, List.any_eq_true, beq_iff_eq, names, and_assoc]

/-- Executable target lookup retains the complete plan and its identity. -/
def findTarget (targets : Targets) (identity : Identity) : Option TargetDeclaration :=
  targets.find? (fun target => target.identity == identity)

theorem findTarget_mem {targets : Targets} {identity : Identity}
    {target : TargetDeclaration} (found : findTarget targets identity = some target) :
    target ∈ targets :=
  List.mem_of_find?_eq_some found

theorem findTarget_identity {targets : Targets} {identity : Identity}
    {target : TargetDeclaration} (found : findTarget targets identity = some target) :
    target.identity = identity := by
  exact beq_iff_eq.mp (List.find?_some
    (p := fun candidate : TargetDeclaration => candidate.identity == identity) found)

theorem findTarget_of_mem {targets : Targets}
    (wellFormed : WellFormedTargets targets) {target : TargetDeclaration}
    (member : target ∈ targets) :
    findTarget targets target.identity = some target :=
  findByIdentity_of_mem TargetDeclaration.identity wellFormed.1 member

/-- A schema matcher/decoder obtains a strict same-wire expansion rank from
its successful target lookup. Consuming edges intentionally do not use it. -/
theorem target_nonconsuming_rank {targets : Targets}
    (wellFormed : WellFormedTargets targets) {identity child : Identity}
    {target : TargetDeclaration} (found : findTarget targets identity = some target)
    (edge : child ∈ targetNonconsumingChildren target.target) :
    ∃ next, findTarget targets child = some next ∧
      next.expansionRank < target.expansionRank := by
  obtain ⟨next, member, sameIdentity, smaller⟩ :=
    (wellFormed.2 target (findTarget_mem found)).1 child edge
  exact ⟨next, sameIdentity ▸ findTarget_of_mem wellFormed member, smaller⟩

/-- Payload descent may return to a parent target. Existence suffices here;
requiring decreasing rank would wrongly reject recursive object/array models. -/
theorem target_consuming_exists {targets : Targets}
    (wellFormed : WellFormedTargets targets) {identity child : Identity}
    {target : TargetDeclaration} (found : findTarget targets identity = some target)
    (edge : child ∈ targetConsumingChildren target.target) :
    ∃ next, findTarget targets child = some next := by
  obtain ⟨next, member, sameIdentity⟩ :=
    (wellFormed.2 target (findTarget_mem found)).2.1 child edge
  exact ⟨next, sameIdentity ▸ findTarget_of_mem wellFormed member⟩

/-- Non-vacuity: recursive collection declarations are accepted at every rank.
The checker imposes no inhabitant-depth bound on the values traversing them. -/
theorem recursiveArrayDeclarationAccepted (rank : Nat) :
    validateDeclarations [
      { identity := ⟨0, 0⟩, expansionRank := rank,
        contract := .array ⟨0, 0⟩ {} }] = true := by
  simp [validateDeclarations, nonconsumingChildren, consumingChildren, declarationNamesValid]

/-- Non-vacuity for the wire guard: object descent resets rank before the
selector returns to the object. The valid recursive plan is not a raw cycle. -/
theorem recursiveObjectSelectorAccepted :
    validateTargets [
      { identity := ⟨0, 0⟩, expansionRank := 0,
        target := .object [
          { identity := ⟨0, 2⟩, wireName := "child", required := false,
            presence := .explicit, child := ⟨0, 1⟩ }] false },
      { identity := ⟨0, 1⟩, expansionRank := 1,
          target := .select ⟨0, 2⟩ ⟨0, 0⟩ }] = true := by decide

/-- Source/wire aliases may overlap across different members. Rejecting that
supported case would remove the ambiguity/collision problem from the domain. -/
theorem crossNamespaceAliasOverlapAccepted :
    validateDeclarations [
      { identity := ⟨0, 0⟩, expansionRank := 0,
        contract := .object [
          { identity := ⟨0, 1⟩, sourceName := "left", wireAlias := "right",
            required := false, child := ⟨0, 3⟩ },
          { identity := ⟨0, 2⟩, sourceName := "right", wireAlias := "other",
            required := false, child := ⟨0, 3⟩ }] false },
      { identity := ⟨0, 3⟩, expansionRank := 0, contract := .any }] = true := by decide

/-- A missing child is rejected rather than treated as an unconstrained type. -/
theorem missingDeclarationChildRejected :
    validateDeclarations [
      { identity := ⟨0, 0⟩, expansionRank := 0,
        contract := .array ⟨0, 1⟩ {} }] = false := by decide

/-- A selector cannot establish progress merely by selecting a semantic field:
its wire/schema recursion sees the same wire, so a raw cycle is rejected. -/
theorem rawSelectorCycleRejected :
    validateTargets [
      { identity := ⟨0, 0⟩, expansionRank := 0,
        target := .select ⟨0, 1⟩ ⟨0, 0⟩ }] = false := by decide

/-- Untagged alternatives see the same wire and cannot recurse at equal rank. -/
theorem rawUntaggedCycleRejected :
    validateTargets [
      { identity := ⟨0, 0⟩, expansionRank := 0,
        target := .union ⟨0, 1⟩ .untagged [
        { identity := ⟨0, 2⟩, wireName := "again", child := ⟨0, 0⟩ }] }] = false := by decide

/-- Tagged envelopes consume a payload, permitting recursive alternatives. -/
theorem recursiveTaggedUnionAccepted :
    validateTargets [
      { identity := ⟨0, 0⟩, expansionRank := 0,
        target := .union ⟨0, 1⟩ (.tagged "kind" "value") [
        { identity := ⟨0, 2⟩, wireName := "again", child := ⟨0, 0⟩ }] }] = true := by decide

/-- An envelope cannot assign its discriminator and payload to the same key. -/
theorem collidingTaggedKeysRejected :
    validateTargets [
      { identity := ⟨0, 0⟩, expansionRank := 0,
        target := .union ⟨0, 1⟩ (.tagged "value" "value") [
        { identity := ⟨0, 2⟩, wireName := "again", child := ⟨0, 0⟩ }] }] = false := by decide

/-- Required element wrappers expand at the same input and wire node. A valid
child decreases rank; a missing child cannot pass either graph checker. -/
theorem nonNullWrapperRankChecked :
    validateDeclarations [
      { identity := ⟨0, 0⟩, expansionRank := 1, contract := .nonNull ⟨0, 1⟩ },
      { identity := ⟨0, 1⟩, expansionRank := 0, contract := .any }] = true ∧
    validateTargets [
      { identity := ⟨0, 0⟩, expansionRank := 1, target := .nonNull ⟨0, 1⟩ },
      { identity := ⟨0, 1⟩, expansionRank := 0, target := .any }] = true ∧
    validateDeclarations [
      { identity := ⟨0, 0⟩, expansionRank := 1, contract := .nonNull ⟨0, 1⟩ }] = false ∧
    validateTargets [
      { identity := ⟨0, 0⟩, expansionRank := 1, target := .nonNull ⟨0, 0⟩ }] = false := by decide

end ValueContract.Candidate
