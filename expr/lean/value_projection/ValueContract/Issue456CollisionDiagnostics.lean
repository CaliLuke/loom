import ValueContract
import ValueContract.NumericCoercionControls

namespace ValueContract.Candidate.Issue456CollisionDiagnostics

structure CollisionWitness where
  path : List String
  branches : List Identity
  name : String
  deriving DecidableEq, Repr

structure CollisionEvidence where
  unknown : Bool
  witnesses : List CollisionWitness
  deriving DecidableEq, Repr

def CollisionEvidence.clear : CollisionEvidence := ⟨false, []⟩
def CollisionEvidence.unknownOnly : CollisionEvidence := ⟨true, []⟩
def CollisionEvidence.proven (witness : CollisionWitness) : CollisionEvidence := ⟨false, [witness]⟩

def combineContainer (children : List CollisionEvidence) : CollisionEvidence :=
  ⟨children.any (·.unknown), children.flatMap (·.witnesses)⟩

theorem combineContainer_retains_every_witness (children : List CollisionEvidence) :
    (combineContainer children).witnesses = children.flatMap (·.witnesses) := by
  rfl

theorem combineContainer_collision_survives_unknown (witness : CollisionWitness) :
    (combineContainer [.unknownOnly, .proven witness]).witnesses = [witness] := by
  rfl

def knownNames (codec : KeyCodec) (keys : List Scalar) : List String :=
  keys.filterMap (encodeKey codec)

def duplicateNames (codec : KeyCodec) (keys : List Scalar) : List String :=
  let names := knownNames codec keys
  (names.filter fun name => decide (2 ≤ names.count name)).eraseDups

def hasUnknownName (codec : KeyCodec) (keys : List Scalar) : Bool :=
  keys.any fun key => (encodeKey codec key).isNone

def collisionEvidenceFromNames (path : List String) (branches : List Identity)
    (names : List (Option String)) : CollisionEvidence :=
  let known := names.filterMap id
  ⟨names.any Option.isNone,
    ((known.filter fun name => decide (2 ≤ known.count name)).eraseDups).map fun name =>
      ⟨path, branches, name⟩⟩

def mapCollisionEvidence (path : List String) (branches : List Identity)
    (codec : KeyCodec) (keys : List Scalar) : CollisionEvidence :=
  collisionEvidenceFromNames path branches (keys.map (encodeKey codec))

theorem duplicateNames_mem_iff_declared_name_count (codec : KeyCodec) (keys : List Scalar)
    (name : String) :
    name ∈ duplicateNames codec keys ↔ 2 ≤ (knownNames codec keys).count name := by
  simp only [duplicateNames, List.mem_eraseDups, List.mem_filter, decide_eq_true_eq]
  constructor
  · exact fun present => present.2
  · intro counted
    exact ⟨List.count_pos_iff.mp (by omega), counted⟩

theorem collision_witness_iff_known_name_count (path : List String)
    (branches : List Identity) (names : List (Option String)) (witness : CollisionWitness) :
    witness ∈ (collisionEvidenceFromNames path branches names).witnesses ↔
      witness.path = path ∧ witness.branches = branches ∧
        2 ≤ (names.filterMap id).count witness.name := by
  simp only [collisionEvidenceFromNames, List.mem_map, List.mem_eraseDups, List.mem_filter,
    decide_eq_true_eq]
  constructor
  · rintro ⟨name, ⟨_namePresent, counted⟩, rfl⟩
    exact ⟨rfl, rfl, counted⟩
  · rintro ⟨pathEq, branchesEq, counted⟩
    refine ⟨witness.name, ⟨?_, counted⟩, ?_⟩
    · exact List.count_pos_iff.mp (by omega)
    · cases witness
      simp_all

theorem map_collision_witness_iff_declared_name_count (path : List String)
    (branches : List Identity) (codec : KeyCodec) (keys : List Scalar)
    (witness : CollisionWitness) :
    witness ∈ (mapCollisionEvidence path branches codec keys).witnesses ↔
      witness.path = path ∧ witness.branches = branches ∧
        2 ≤ (knownNames codec keys).count witness.name := by
  simpa [mapCollisionEvidence, knownNames] using
    collision_witness_iff_known_name_count path branches (keys.map (encodeKey codec)) witness

theorem map_collision_witness_membership_permutation_invariant (path : List String)
    (branches : List Identity) (codec : KeyCodec) {left right : List Scalar}
    (permuted : left.Perm right) (witness : CollisionWitness) :
    witness ∈ (mapCollisionEvidence path branches codec left).witnesses ↔
      witness ∈ (mapCollisionEvidence path branches codec right).witnesses := by
  rw [map_collision_witness_iff_declared_name_count,
    map_collision_witness_iff_declared_name_count]
  have namesPerm : (knownNames codec left).Perm (knownNames codec right) := by
    exact permuted.filterMap (encodeKey codec)
  rw [namesPerm.count_eq witness.name]

theorem map_collision_uses_declared_names (path : List String) (branches : List Identity)
    (codec : KeyCodec) (keys : List Scalar) :
    mapCollisionEvidence path branches codec keys =
      collisionEvidenceFromNames path branches (keys.map (encodeKey codec)) := by
  rfl

theorem unsupported_name_is_unknown :
    collisionEvidenceFromNames [] [] [none] =
      .unknownOnly := by
  rfl

theorem known_collision_survives_unsupported_key :
    (collisionEvidenceFromNames [] [] [some "1", none, some "1"]).witnesses =
      [⟨[], [], "1"⟩] := by
  rfl

theorem heterogeneous_collision_control :
    collisionEvidenceFromNames ["field"] [] [some "1", some "1"] =
      ⟨false, [⟨["field"], [], "1"⟩]⟩ := by
  rfl

theorem declared_float32_collision_control :
    resolve NumericCoercionControls.mapDeclarations ResolutionControls.checks
      NumericCoercionControls.numbers .authoredExample ResolutionControls.oid
      (.map [(NumericCoercionControls.narrowTenth, .scalar (.string "first")),
        (NumericRepresentationControls.binaryTenth .binary64,
          .scalar (.string "second"))]) = .error .invalid := by
  exact NumericCoercionControls.declaredPrecisionCollisionRejected

theorem raw_float_values_are_distinct :
    scalarEqual NumericCoercionControls.narrowTenth
      NumericCoercionControls.wideTenth = false := by
  exact NumericCoercionControls.exactOnlyCoercionMissesNormalization.2

theorem raw_float_key_spellings_are_distinct :
    encodeKey NumericCoercionControls.numbers NumericCoercionControls.narrowTenth =
        some "0.1" ∧
      encodeKey NumericCoercionControls.numbers NumericCoercionControls.wideTenth =
        some "0.10000000149011612" := by
  decide

structure IgnoredPredicates where
  length : Nat
  required : Bool
  enumeration : List Scalar
  deriving DecidableEq, Repr

def inspectMap (codec : KeyCodec) (keys : List Scalar) (_ : IgnoredPredicates) : CollisionEvidence :=
  mapCollisionEvidence [] [] codec keys

theorem collision_predicates_irrelevant (codec : KeyCodec) (keys : List Scalar)
    (left right : IgnoredPredicates) :
    inspectMap codec keys left = inspectMap codec keys right := by
  rfl

inductive Compatibility where
  | incompatible
  | compatible
  | unknown
  deriving DecidableEq, Repr

structure AlternativeEvidence where
  identity : Identity
  eligibility : Compatibility
  collision : CollisionEvidence
  deriving DecidableEq, Repr

def UnionCollisionClaim (alternatives : List AlternativeEvidence) : Prop :=
  (∀ alternative ∈ alternatives, alternative.eligibility ≠ .unknown) ∧
  (∃ alternative ∈ alternatives, alternative.eligibility = .compatible) ∧
  (∀ alternative ∈ alternatives, alternative.eligibility = .compatible →
    alternative.collision.witnesses ≠ [])

def unionRejects (alternatives : List AlternativeEvidence) : Bool :=
  (alternatives.all fun alternative => decide (alternative.eligibility ≠ .unknown)) &&
    (alternatives.any fun alternative => decide (alternative.eligibility = .compatible)) &&
      (alternatives.all fun alternative => decide (alternative.eligibility = .compatible →
        alternative.collision.witnesses ≠ []))

def unionCollisionWitnesses (alternatives : List AlternativeEvidence) :
    Option (List (Identity × List CollisionWitness)) :=
  if unionRejects alternatives then
    some ((alternatives.filter (·.eligibility == .compatible)).map fun alternative =>
      (alternative.identity, alternative.collision.witnesses))
  else none

theorem unionRejects_iff (alternatives : List AlternativeEvidence) :
    unionRejects alternatives = true ↔ UnionCollisionClaim alternatives := by
  simp [unionRejects, UnionCollisionClaim, and_assoc, Decidable.imp_iff_not_or]

theorem union_collision_witness_branch_identity_sound (alternatives : List AlternativeEvidence)
    (result : List (Identity × List CollisionWitness)) (identity : Identity)
    (witnesses : List CollisionWitness)
    (returned : unionCollisionWitnesses alternatives = some result)
    (present : (identity, witnesses) ∈ result) :
    ∃ alternative ∈ alternatives,
      alternative.eligibility = .compatible ∧
      alternative.identity = identity ∧
      alternative.collision.witnesses = witnesses := by
  unfold unionCollisionWitnesses at returned
  split at returned
  · simp only [Option.some.injEq] at returned
    subst result
    simp only [List.mem_map, List.mem_filter, beq_iff_eq] at present
    rcases present with ⟨alternative, ⟨inAlternatives, compatible⟩, pairEq⟩
    exact ⟨alternative, inAlternatives, compatible, Prod.mk.inj pairEq |>.1,
      Prod.mk.inj pairEq |>.2⟩
  · simp at returned

def selectedCollisionEvidence (branch : Identity) (alternatives : List AlternativeEvidence) :
    Option CollisionEvidence :=
  (alternatives.find? fun alternative => alternative.identity == branch).map (·.collision)

def branchA : Identity := ⟨456, 1⟩
def branchB : Identity := ⟨456, 2⟩
def invalidBranch : Identity := ⟨456, 3⟩
def witnessA : CollisionWitness := ⟨["a"], [branchA], "1"⟩
def witnessB : CollisionWitness := ⟨["b"], [branchB], "true"⟩

def collisionAlternative : AlternativeEvidence := ⟨branchA, .compatible, .proven witnessA⟩
def clearAlternative : AlternativeEvidence := ⟨branchB, .compatible, .clear⟩
def unknownCollisionAlternative : AlternativeEvidence := ⟨branchB, .compatible, .unknownOnly⟩
def unknownEligibilityAlternative : AlternativeEvidence := ⟨branchB, .unknown, .clear⟩
def incompatibleAlternative : AlternativeEvidence := ⟨branchB, .incompatible, .clear⟩

theorem compatible_clear_blocks_union_rejection :
    unionCollisionWitnesses [collisionAlternative, clearAlternative] = none := by
  classical
  simp [unionCollisionWitnesses, unionRejects,
    collisionAlternative, clearAlternative, CollisionEvidence.proven,
    CollisionEvidence.clear]

theorem compatible_unknown_collision_blocks_union_rejection :
    unionCollisionWitnesses [collisionAlternative, unknownCollisionAlternative] = none := by
  classical
  simp [unionCollisionWitnesses, unionRejects,
    collisionAlternative, unknownCollisionAlternative, CollisionEvidence.proven,
    CollisionEvidence.unknownOnly]

theorem unknown_eligibility_blocks_universal_collision_claim :
    unionCollisionWitnesses [collisionAlternative, unknownEligibilityAlternative] = none := by
  classical
  simp [unionCollisionWitnesses, unionRejects,
    collisionAlternative, unknownEligibilityAlternative]

theorem zero_compatible_alternatives_defer :
    unionCollisionWitnesses [{collisionAlternative with eligibility := .incompatible},
      incompatibleAlternative] = none := by
  classical
  simp [unionCollisionWitnesses, unionRejects,
    collisionAlternative, incompatibleAlternative]

theorem every_compatible_collision_retains_branch_witnesses :
    unionCollisionWitnesses [collisionAlternative,
      ⟨branchB, .compatible, .proven witnessB⟩] =
      some [(branchA, [witnessA]), (branchB, [witnessB])] := by
  classical
  simp [unionCollisionWitnesses, unionRejects,
    collisionAlternative, branchA, branchB, witnessA, witnessB,
    CollisionEvidence.proven]

theorem incompatible_alternative_does_not_rescue_collision :
    unionCollisionWitnesses [collisionAlternative, incompatibleAlternative] =
      some [(branchA, [witnessA])] := by
  classical
  simp [unionCollisionWitnesses, unionRejects,
    collisionAlternative, incompatibleAlternative, branchA, witnessA,
    CollisionEvidence.proven]

theorem selected_branch_is_authoritative :
    selectedCollisionEvidence branchA [collisionAlternative, clearAlternative] =
      some (.proven witnessA) ∧
    selectedCollisionEvidence invalidBranch [collisionAlternative, clearAlternative] = none := by
  decide

def objectSourceCollision (entries : List (String × Input)) : Bool :=
  !(entries.map Prod.fst).Nodup

def lastObjectValue? (name : String) (entries : List (String × Input)) : Option Input :=
  (entries.reverse.find? fun entry => entry.1 == name).map Prod.snd

def duplicateObjectEntries : List (String × Input) :=
  [("items", .scalar ⟨.string "first", .builtinString⟩),
    ("items", .scalar ⟨.string "second", .builtinString⟩)]

theorem duplicate_object_names_rejected_before_insertion :
    objectSourceCollision duplicateObjectEntries = true := by
  rfl

theorem right_biased_object_insertion_loses_a_duplicate :
    lastObjectValue? "items" duplicateObjectEntries =
        some (.scalar ⟨.string "second", .builtinString⟩) ∧
    lastObjectValue? "items" duplicateObjectEntries.reverse =
        some (.scalar ⟨.string "first", .builtinString⟩) := by
  simp [lastObjectValue?, duplicateObjectEntries]

inductive CompatibilityTree where
  | atom (compatible : Bool)
  | pair (left right : CompatibilityTree)
  | alias (child : CompatibilityTree)
  | cycle
  | opaqueLeaf
  deriving Repr

def combineCompatibility (left right : Compatibility) : Compatibility :=
  if left = .incompatible || right = .incompatible then .incompatible
  else if left = .unknown || right = .unknown then .unknown
  else .compatible

def finiteCompatibility : CompatibilityTree → Compatibility
  | .atom true => .compatible
  | .atom false => .incompatible
  | .pair left right => combineCompatibility (finiteCompatibility left) (finiteCompatibility right)
  | .alias child => finiteCompatibility child
  | .cycle | .opaqueLeaf => .unknown

def guardedCompatibility : Nat → CompatibilityTree → Compatibility
  | 0, _ => .unknown
  | _ + 1, .atom true => .compatible
  | _ + 1, .atom false => .incompatible
  | depth + 1, .pair left right =>
      combineCompatibility (guardedCompatibility depth left) (guardedCompatibility depth right)
  | depth + 1, .alias child => guardedCompatibility depth child
  | _ + 1, .cycle | _ + 1, .opaqueLeaf => .unknown

def compatibilityDepth : CompatibilityTree → Nat
  | .atom _ | .cycle | .opaqueLeaf => 1
  | .alias child => compatibilityDepth child + 1
  | .pair left right => max (compatibilityDepth left) (compatibilityDepth right) + 1

theorem guardedCompatibility_sufficient (tree : CompatibilityTree) (fuel : Nat)
    (enough : compatibilityDepth tree ≤ fuel) :
    guardedCompatibility fuel tree = finiteCompatibility tree := by
  induction tree generalizing fuel with
  | atom compatible =>
      cases compatible <;> cases fuel <;>
        simp_all [compatibilityDepth, guardedCompatibility, finiteCompatibility]
  | cycle => cases fuel <;> simp_all [compatibilityDepth, guardedCompatibility, finiteCompatibility]
  | opaqueLeaf => cases fuel <;> simp_all [compatibilityDepth, guardedCompatibility, finiteCompatibility]
  | alias child ih =>
      cases fuel with
      | zero => simp [compatibilityDepth] at enough
      | succ fuel =>
          simp only [guardedCompatibility, finiteCompatibility]
          apply ih
          change compatibilityDepth child + 1 ≤ fuel + 1 at enough
          exact Nat.le_of_succ_le_succ enough
  | pair left right leftIH rightIH =>
      cases fuel with
      | zero => simp [compatibilityDepth] at enough
      | succ fuel =>
          simp only [guardedCompatibility, finiteCompatibility]
          change max (compatibilityDepth left) (compatibilityDepth right) + 1 ≤ fuel + 1 at enough
          have maxBound := Nat.le_of_succ_le_succ enough
          rw [leftIH fuel (Nat.le_trans (Nat.le_max_left _ _) maxBound),
            rightIH fuel (Nat.le_trans (Nat.le_max_right _ _) maxBound)]

theorem guardedCompatibility_at_derived_bound (tree : CompatibilityTree) :
    guardedCompatibility (compatibilityDepth tree) tree = finiteCompatibility tree := by
  exact guardedCompatibility_sufficient tree _ (Nat.le_refl _)

def stringLE (left right : String) : Bool := decide (left ≤ right)

def listLexLE (le : α → α → Bool) : List α → List α → Bool
  | [], _ => true
  | _ :: _, [] => false
  | left :: lefts, right :: rights =>
      if le left right then if le right left then listLexLE le lefts rights else true else false

def identityLE (left right : Identity) : Bool :=
  left.occurrence < right.occurrence ||
    (left.occurrence == right.occurrence && left.declaration ≤ right.declaration)

def witnessLE (left right : CollisionWitness) : Bool :=
  if left.path == right.path then
    if left.branches == right.branches then stringLE left.name right.name
    else listLexLE identityLE left.branches right.branches
  else listLexLE stringLE left.path right.path

def insertWitness (witness : CollisionWitness) : List CollisionWitness → List CollisionWitness
  | [] => [witness]
  | head :: tail =>
      if witnessLE witness head then witness :: head :: tail
      else head :: insertWitness witness tail

def sortWitnesses : List CollisionWitness → List CollisionWitness
  | [] => []
  | head :: tail => insertWitness head (sortWitnesses tail)

def presentCollisions (evidence : CollisionEvidence) : List CollisionWitness :=
  sortWitnesses evidence.witnesses

def pathFirst : CollisionWitness := ⟨["a"], [branchB], "z"⟩
def nameFirst : CollisionWitness := ⟨["z"], [branchA], "a"⟩

theorem presentation_uses_structural_path_before_name :
    presentCollisions ⟨false, [nameFirst, pathFirst]⟩ = [pathFirst, nameFirst] := by
  rfl

theorem structural_path_components_do_not_collapse :
    ({path := ["a.b", "c"], branches := [branchA], name := "1"} : CollisionWitness) ≠
      {path := ["a", "b.c"], branches := [branchA], name := "1"} := by
  decide

end ValueContract.Candidate.Issue456CollisionDiagnostics
