import ValueContract.NumericBounds

namespace ValueContract.Candidate

/-- Pattern and format clauses have disjoint identities even when their
numeric predicate IDs coincide. The IDs are supplied by an independent
adapter; this model does not interpret regular expressions or formats. -/
inductive PredicateKind where
  | pattern
  | format
  deriving BEq, ReflBEq, LawfulBEq, DecidableEq, Repr

structure PredicateIdentity where
  kind : PredicateKind
  predicate : Nat
  deriving BEq, ReflBEq, LawfulBEq, DecidableEq, Repr

structure AuthoredPredicateClause where
  kind : PredicateKind
  predicate : Nat
  origin : Nat
  deriving BEq, ReflBEq, LawfulBEq, DecidableEq, Repr

def AuthoredPredicateClause.identity (clause : AuthoredPredicateClause) :
    PredicateIdentity :=
  { kind := clause.kind, predicate := clause.predicate }

/-- One independently resolved authored value. `semantic` names an equivalence
class produced by declared-type resolution; the model never compares raw JSON. -/
structure AuthoredContractValue where
  origin : Nat
  semantic : Nat
  number : Option Decimal := none
  satisfiedPredicates : List PredicateIdentity := []
  deriving BEq, DecidableEq, Repr

/-- A finalized field identity is separate from the place that required it. -/
structure RequiredField where
  declaration : Nat
  field : Nat
  origin : Nat
  deriving BEq, DecidableEq, Repr

def RequiredField.sameIdentity (left right : RequiredField) : Bool :=
  left.declaration == right.declaration && left.field == right.field

def RequiredField.identity (field : RequiredField) : Nat × Nat :=
  (field.declaration, field.field)

/-- One raw authored declaration layer, ordered from oldest ancestor to the
effective occurrence. Absence remains distinct from an explicitly present list
or default. -/
structure AliasContractLayer where
  declaration : Nat
  enumeration : Option (List AuthoredContractValue) := none
  defaultValue : Option AuthoredContractValue := none
  numeric : AuthoredNumericBounds := {}
  predicates : List AuthoredPredicateClause := []
  required : List RequiredField := []
  deriving BEq, DecidableEq, Repr

structure EffectiveAliasContract where
  enumeration : Option (List AuthoredContractValue)
  defaultValue : Option AuthoredContractValue
  numericLayers : List AuthoredNumericBounds
  predicates : List AuthoredPredicateClause
  required : List RequiredField
  deriving BEq, DecidableEq, Repr

inductive AliasContractError where
  | enumWidening (layer declaration offendingOrigin offendingSemantic : Nat)
  | invalidEnum (origin semantic : Nat)
  | invalidDefault (origin semantic : Nat)
  deriving BEq, DecidableEq, Repr

def semanticEqual (left right : AuthoredContractValue) : Bool :=
  left.semantic == right.semantic

def enumContains (enumeration : List AuthoredContractValue)
    (value : AuthoredContractValue) : Bool :=
  enumeration.any (semanticEqual value)

def enumRefines (parent child : List AuthoredContractValue) : Prop :=
  ∀ value ∈ child, enumContains parent value = true

def firstEnumWidening (parent child : List AuthoredContractValue) :
    Option AuthoredContractValue :=
  child.find? fun value => !enumContains parent value

/-- Legacy behavior selected the last explicit enum without checking ancestry. -/
def legacyEffectiveEnum (layers : List AliasContractLayer) :
    Option (List AuthoredContractValue) :=
  layers.foldl (fun current layer => layer.enumeration.or current) none

private def applyEnumeration (layerIndex : Nat)
    (current : Option (List AuthoredContractValue)) (layer : AliasContractLayer) :
    Except AliasContractError (Option (List AuthoredContractValue)) :=
  match current, layer.enumeration with
  | _, none => .ok current
  | none, some authored => .ok (some authored)
  | some parent, some authored =>
      match firstEnumWidening parent authored with
      | none => .ok (some authored)
      | some value => .error (.enumWidening layerIndex layer.declaration
          value.origin value.semantic)

private def effectiveEnumerationFrom (index : Nat)
    (current : Option (List AuthoredContractValue)) : List AliasContractLayer →
    Except AliasContractError (Option (List AuthoredContractValue))
  | [] => .ok current
  | layer :: rest => match applyEnumeration index current layer with
      | .error error => .error error
      | .ok next => effectiveEnumerationFrom (index + 1) next rest

def effectiveEnumeration (layers : List AliasContractLayer) :
    Except AliasContractError (Option (List AuthoredContractValue)) :=
  effectiveEnumerationFrom 0 none layers

private def enumerationSucceeded :
    Except AliasContractError (Option (List AuthoredContractValue)) → Bool
  | .ok _ => true
  | .error _ => false

/-- Independent recursive refinement judgment over the complete raw ancestry. -/
def EnumAncestryValid : Option (List AuthoredContractValue) →
    List AliasContractLayer → Prop
  | _, [] => True
  | current, layer :: rest => match current, layer.enumeration with
      | _, none => EnumAncestryValid current rest
      | none, some authored => EnumAncestryValid (some authored) rest
      | some parent, some authored =>
          enumRefines parent authored ∧ EnumAncestryValid (some authored) rest

theorem firstEnumWidening_none_iff (parent authored : List AuthoredContractValue) :
    firstEnumWidening parent authored = none ↔ enumRefines parent authored := by
  simp [firstEnumWidening, enumRefines]

private theorem effectiveEnumerationFrom_valid (index current layers) :
    enumerationSucceeded (effectiveEnumerationFrom index current layers) = true ↔
      EnumAncestryValid current layers := by
  induction layers generalizing index current with
  | nil => simp [effectiveEnumerationFrom, enumerationSucceeded, EnumAncestryValid]
  | cons layer rest ih =>
    cases current with
    | none =>
      cases hauthored : layer.enumeration with
      | none =>
        simp only [effectiveEnumerationFrom, applyEnumeration, EnumAncestryValid, hauthored]
        exact ih (index + 1) none
      | some authored =>
        simp only [effectiveEnumerationFrom, applyEnumeration, EnumAncestryValid, hauthored]
        exact ih (index + 1) (some authored)
    | some parent =>
      cases hauthored : layer.enumeration with
      | none =>
        simp only [effectiveEnumerationFrom, applyEnumeration, EnumAncestryValid, hauthored]
        exact ih (index + 1) (some parent)
      | some authored =>
        cases hwidening : firstEnumWidening parent authored with
        | none =>
          have valid := (firstEnumWidening_none_iff parent authored).1 hwidening
          simp only [effectiveEnumerationFrom, applyEnumeration, EnumAncestryValid,
            hauthored, hwidening]
          simp only [valid, true_and]
          exact ih (index + 1) (some authored)
        | some offending =>
          have invalid : ¬enumRefines parent authored := by
            intro valid
            have hnone := (firstEnumWidening_none_iff parent authored).2 valid
            rw [hwidening] at hnone
            contradiction
          simp [effectiveEnumerationFrom, applyEnumeration, EnumAncestryValid,
            enumerationSucceeded, hauthored, hwidening, invalid]

/-- Candidate enum acceptance is exactly the independent raw-ancestry
refinement judgment, for all resolved semantic classes and layer counts. -/
theorem effectiveEnumeration_iff_raw_refinement (layers : List AliasContractLayer) :
    enumerationSucceeded (effectiveEnumeration layers) = true ↔
      EnumAncestryValid none layers := by
  exact effectiveEnumerationFrom_valid 0 none layers

private theorem effectiveEnumerationFrom_returns_last (index current layers)
    (valid : EnumAncestryValid current layers) :
    effectiveEnumerationFrom index current layers = .ok
      (layers.foldl (fun prior layer => layer.enumeration.or prior) current) := by
  induction layers generalizing index current with
  | nil => simp [effectiveEnumerationFrom]
  | cons layer rest ih =>
    cases current with
    | none =>
      cases hauthored : layer.enumeration with
      | none =>
        simp only [EnumAncestryValid, hauthored] at valid
        simp only [effectiveEnumerationFrom, applyEnumeration, hauthored, List.foldl_cons]
        exact ih (index + 1) none valid
      | some authored =>
        simp only [EnumAncestryValid, hauthored] at valid
        simp only [effectiveEnumerationFrom, applyEnumeration, hauthored, List.foldl_cons]
        exact ih (index + 1) (some authored) valid
    | some parent =>
      cases hauthored : layer.enumeration with
      | none =>
        simp only [EnumAncestryValid, hauthored] at valid
        simp only [effectiveEnumerationFrom, applyEnumeration, hauthored, List.foldl_cons]
        exact ih (index + 1) (some parent) valid
      | some authored =>
        simp only [EnumAncestryValid, hauthored] at valid
        have noWidening := (firstEnumWidening_none_iff parent authored).2 valid.1
        simp only [effectiveEnumerationFrom, applyEnumeration, hauthored, noWidening,
          List.foldl_cons]
        exact ih (index + 1) (some authored) valid.2

/-- On every valid raw ancestry the candidate returns exactly the last
explicit declaration, preserving absence when no declaration exists. -/
theorem effectiveEnumeration_returns_last_explicit (layers : List AliasContractLayer)
    (valid : EnumAncestryValid none layers) :
    effectiveEnumeration layers = .ok (legacyEffectiveEnum layers) := by
  exact effectiveEnumerationFrom_returns_last 0 none layers valid

def selectedDefault (layers : List AliasContractLayer) : Option AuthoredContractValue :=
  layers.foldl (fun current layer => layer.defaultValue.or current) none

theorem selectedDefault_append (layers : List AliasContractLayer)
    (layer : AliasContractLayer) :
    selectedDefault (layers ++ [layer]) = layer.defaultValue.or (selectedDefault layers) := by
  simp [selectedDefault, List.foldl_append]

/-- A present local default replaces every inherited default. -/
theorem selectedDefault_local_replaces (layers : List AliasContractLayer)
    (layer : AliasContractLayer) (value : AuthoredContractValue)
    (authored : layer.defaultValue = some value) :
    selectedDefault (layers ++ [layer]) = some value := by
  simp [selectedDefault_append, authored]

/-- An absent local default preserves the effective inherited default. -/
theorem selectedDefault_absent_inherits (layers : List AliasContractLayer)
    (layer : AliasContractLayer) (absent : layer.defaultValue = none) :
    selectedDefault (layers ++ [layer]) = selectedDefault layers := by
  simp [selectedDefault_append, absent]

def numericLayersAllow (layers : List AuthoredNumericBounds)
    (value : AuthoredContractValue) : Bool :=
  value.number.all fun number =>
    layers.all fun bounds => numberAllowed (effectiveNumericBounds bounds) number

def mergePredicates (current : List AuthoredPredicateClause) :
    List AuthoredPredicateClause → List AuthoredPredicateClause
  | [] => current
  | clause :: rest =>
      if clause.identity ∈ current.map AuthoredPredicateClause.identity then
        mergePredicates current rest
      else
        mergePredicates (current ++ [clause]) rest

/-- Raw layers are stored base-to-current for enum refinement. Predicate
clauses deliberately traverse the reverse order: current declaration first,
then successive named bases. Exact duplicate identities retain the first,
current-most provenance. -/
def effectivePredicates (layers : List AliasContractLayer) :
    List AuthoredPredicateClause :=
  layers.reverse.foldl (fun current layer => mergePredicates current layer.predicates) []

def predicateClausesAllow (clauses : List AuthoredPredicateClause)
    (value : AuthoredContractValue) : Bool :=
  clauses.all fun clause => value.satisfiedPredicates.contains clause.identity

def effectiveContractAllows (enumeration : Option (List AuthoredContractValue))
    (numeric : List AuthoredNumericBounds) (predicates : List AuthoredPredicateClause)
    (value : AuthoredContractValue) : Bool :=
  enumeration.all (fun values => enumContains values value) &&
    numericLayersAllow numeric value && predicateClausesAllow predicates value

def nonEnumContractAllows (numeric : List AuthoredNumericBounds)
    (predicates : List AuthoredPredicateClause) (value : AuthoredContractValue) : Bool :=
  numericLayersAllow numeric value && predicateClausesAllow predicates value

def firstInvalidEffectiveEnum (enumeration : Option (List AuthoredContractValue))
    (numeric : List AuthoredNumericBounds) (predicates : List AuthoredPredicateClause) :
    Option AuthoredContractValue :=
  enumeration.bind fun values =>
    values.find? fun value => !nonEnumContractAllows numeric predicates value

theorem firstInvalidEffectiveEnum_none_iff
    (enumeration : Option (List AuthoredContractValue))
    (numeric : List AuthoredNumericBounds) (predicates : List AuthoredPredicateClause) :
    firstInvalidEffectiveEnum enumeration numeric predicates = none ↔
      ∀ values ∈ enumeration, ∀ value ∈ values,
        nonEnumContractAllows numeric predicates value = true := by
  cases enumeration with
  | none => simp [firstInvalidEffectiveEnum]
  | some values =>
    simp [firstInvalidEffectiveEnum, List.find?_eq_none]

theorem mergePredicates_identity_mem (current authored : List AuthoredPredicateClause)
    (identity : PredicateIdentity) :
    identity ∈ (mergePredicates current authored).map AuthoredPredicateClause.identity ↔
      identity ∈ current.map AuthoredPredicateClause.identity ∨
        identity ∈ authored.map AuthoredPredicateClause.identity := by
  induction authored generalizing current with
  | nil => simp [mergePredicates]
  | cons clause rest ih =>
    simp only [mergePredicates]
    split
    next present =>
      rw [ih]
      simp only [List.map_cons, List.mem_cons]
      constructor
      · intro found
        exact found.elim Or.inl fun tail => Or.inr (Or.inr tail)
      · intro found
        exact found.elim Or.inl fun authored => authored.elim
          (fun equal => Or.inl (equal ▸ present)) Or.inr
    next absent =>
      rw [ih]
      simp only [List.map_append, List.map_cons, List.mem_append, List.mem_cons]
      grind

theorem mergePredicates_first_identity (current authored : List AuthoredPredicateClause)
    (identity : PredicateIdentity) :
    (mergePredicates current authored).find? (fun clause => clause.identity == identity) =
      (current.find? (fun clause => clause.identity == identity)).or
        (authored.find? (fun clause => clause.identity == identity)) := by
  induction authored generalizing current with
  | nil => simp [mergePredicates]
  | cons clause rest ih =>
    simp only [mergePredicates]
    split
    next present =>
      rw [ih]
      simp only [List.find?_cons]
      split
      next same =>
        cases found : current.find? (fun candidate => candidate.identity == identity) with
        | none =>
          have noneMatch := List.find?_eq_none.mp found
          obtain ⟨candidate, member, equal⟩ := List.mem_map.mp present
          have different := noneMatch candidate member
          simp_all
        | some candidate => simp
      next different => simp
    next absent =>
      rw [ih]
      simp only [List.find?_append, List.find?_cons]
      split <;> simp_all

theorem mergePredicates_identities_nodup (current authored : List AuthoredPredicateClause)
    (unique : (current.map AuthoredPredicateClause.identity).Nodup) :
    ((mergePredicates current authored).map AuthoredPredicateClause.identity).Nodup := by
  induction authored generalizing current with
  | nil => simpa [mergePredicates] using unique
  | cons clause rest ih =>
    simp only [mergePredicates]
    split
    next present => exact ih current unique
    next absent =>
      apply ih (current ++ [clause])
      simpa [List.nodup_append] using And.intro unique absent

private theorem foldPredicates_identities_nodup (ordered : List AliasContractLayer)
    (current : List AuthoredPredicateClause)
    (unique : (current.map AuthoredPredicateClause.identity).Nodup) :
    ((ordered.foldl (fun result layer => mergePredicates result layer.predicates) current).map
      AuthoredPredicateClause.identity).Nodup := by
  induction ordered generalizing current with
  | nil => simpa using unique
  | cons layer rest ih =>
    simp only [List.foldl_cons]
    exact ih (mergePredicates current layer.predicates)
      (mergePredicates_identities_nodup current layer.predicates unique)

theorem effectivePredicates_identities_nodup (layers : List AliasContractLayer) :
    ((effectivePredicates layers).map AuthoredPredicateClause.identity).Nodup := by
  apply foldPredicates_identities_nodup
  simp

private theorem foldPredicates_first_identity (ordered : List AliasContractLayer)
    (current : List AuthoredPredicateClause) (identity : PredicateIdentity) :
    (ordered.foldl (fun result layer => mergePredicates result layer.predicates) current).find?
        (fun clause => clause.identity == identity) =
      (current.find? (fun clause => clause.identity == identity)).or
        ((ordered.flatMap (·.predicates)).find?
          (fun clause => clause.identity == identity)) := by
  induction ordered generalizing current with
  | nil => simp
  | cons layer rest ih =>
    simp only [List.foldl_cons, List.flatMap_cons, ih, mergePredicates_first_identity,
      List.find?_append]
    cases current.find? (fun clause => clause.identity == identity) <;>
      cases layer.predicates.find? (fun clause => clause.identity == identity) <;> simp

/-- Every clause identity retains the first provenance in current-to-base raw
order. -/
theorem effectivePredicates_first_identity (layers : List AliasContractLayer)
    (identity : PredicateIdentity) :
    (effectivePredicates layers).find? (fun clause => clause.identity == identity) =
      ((layers.reverse.flatMap (·.predicates)).find?
        (fun clause => clause.identity == identity)) := by
  simpa [effectivePredicates] using
    foldPredicates_first_identity layers.reverse [] identity

/-- Every independently extracted raw Pattern or Format identity occurs in the
effective clause union. -/
theorem effectivePredicates_contains_raw (layers : List AliasContractLayer)
    (layer : AliasContractLayer) (clause : AuthoredPredicateClause)
    (layerPresent : layer ∈ layers) (clausePresent : clause ∈ layer.predicates) :
    clause.identity ∈
      (effectivePredicates layers).map AuthoredPredicateClause.identity := by
  have reversed : layer ∈ layers.reverse := by simpa using layerPresent
  have flattened : clause ∈ layers.reverse.flatMap (·.predicates) :=
    List.mem_flatMap.mpr ⟨layer, reversed, clausePresent⟩
  by_cases present : clause.identity ∈
      (effectivePredicates layers).map AuthoredPredicateClause.identity
  · exact present
  · exfalso
    have outputNone :
        (effectivePredicates layers).find?
          (fun candidate => candidate.identity == clause.identity) = none := by
      apply List.find?_eq_none.mpr
      intro candidate member equal
      apply present
      exact List.mem_map.mpr ⟨candidate, member, beq_iff_eq.mp equal⟩
    have rawNotNone :
        (layers.reverse.flatMap (·.predicates)).find?
          (fun candidate => candidate.identity == clause.identity) ≠ none := by
      intro absent
      have noMatch := List.find?_eq_none.mp absent
      exact (noMatch clause flattened) (by simp)
    rw [effectivePredicates_first_identity] at outputNone
    exact rawNotNone outputNone

def mergeRequired (current : List RequiredField) : List RequiredField → List RequiredField
  | [] => current
  | field :: rest =>
      if field.identity ∈ current.map RequiredField.identity then
        mergeRequired current rest
      else
        mergeRequired (current ++ [field]) rest

def effectiveRequired (layers : List AliasContractLayer) : List RequiredField :=
  layers.reverse.foldl (fun current layer => mergeRequired current layer.required) []

theorem mergeRequired_identity_mem (current authored : List RequiredField)
    (identity : Nat × Nat) :
    identity ∈ (mergeRequired current authored).map RequiredField.identity ↔
      identity ∈ current.map RequiredField.identity ∨
        identity ∈ authored.map RequiredField.identity := by
  induction authored generalizing current with
  | nil => simp [mergeRequired]
  | cons field rest ih =>
    simp only [mergeRequired]
    split
    next present =>
      rw [ih]
      simp only [List.map_cons, List.mem_cons]
      constructor
      · intro found
        exact found.elim Or.inl fun tail => Or.inr (Or.inr tail)
      · intro found
        exact found.elim Or.inl fun authored => authored.elim
          (fun equal => Or.inl (equal ▸ present)) Or.inr
    next absent =>
      rw [ih]
      simp only [List.map_append, List.map_cons, List.mem_append, List.mem_cons]
      grind

theorem mergeRequired_first_identity (current authored : List RequiredField)
    (identity : Nat × Nat) :
    (mergeRequired current authored).find? (fun field => field.identity == identity) =
      (current.find? (fun field => field.identity == identity)).or
        (authored.find? (fun field => field.identity == identity)) := by
  induction authored generalizing current with
  | nil => simp [mergeRequired]
  | cons field rest ih =>
    simp only [mergeRequired]
    split
    next present =>
      rw [ih]
      simp only [List.find?_cons]
      split
      next same =>
        cases found : current.find? (fun candidate => candidate.identity == identity) with
        | none =>
          have noneMatch := List.find?_eq_none.mp found
          obtain ⟨candidate, member, equal⟩ := List.mem_map.mp present
          have different := noneMatch candidate member
          simp_all
        | some candidate => simp
      next different => simp
    next absent =>
      rw [ih]
      simp only [List.find?_append, List.find?_cons]
      split <;> simp_all

theorem mergeRequired_identities_nodup (current authored : List RequiredField)
    (unique : (current.map RequiredField.identity).Nodup) :
    ((mergeRequired current authored).map RequiredField.identity).Nodup := by
  induction authored generalizing current with
  | nil => simpa [mergeRequired] using unique
  | cons field rest ih =>
    simp only [mergeRequired]
    split
    next present => exact ih current unique
    next absent =>
      apply ih (current ++ [field])
      simpa [List.nodup_append] using And.intro unique absent

private theorem foldRequired_identities_nodup (ordered : List AliasContractLayer)
    (current : List RequiredField) (unique : (current.map RequiredField.identity).Nodup) :
    ((ordered.foldl (fun result layer => mergeRequired result layer.required) current).map
      RequiredField.identity).Nodup := by
  induction ordered generalizing current with
  | nil => simpa using unique
  | cons layer rest ih =>
    simp only [List.foldl_cons]
    exact ih (mergeRequired current layer.required)
      (mergeRequired_identities_nodup current layer.required unique)

theorem effectiveRequired_identities_nodup (layers : List AliasContractLayer) :
    ((effectiveRequired layers).map RequiredField.identity).Nodup := by
  apply foldRequired_identities_nodup
  simp

private theorem foldRequired_first_identity (ordered : List AliasContractLayer)
    (current : List RequiredField) (identity : Nat × Nat) :
    (ordered.foldl (fun result layer => mergeRequired result layer.required) current).find?
        (fun field => field.identity == identity) =
      (current.find? (fun field => field.identity == identity)).or
        ((ordered.flatMap (·.required)).find? (fun field => field.identity == identity)) := by
  induction ordered generalizing current with
  | nil => simp
  | cons layer rest ih =>
    simp only [List.foldl_cons, List.flatMap_cons, ih, mergeRequired_first_identity,
      List.find?_append]
    cases current.find? (fun field => field.identity == identity) <;>
      cases layer.required.find? (fun field => field.identity == identity) <;> simp

/-- Requiredness retains the first raw provenance record in effective-to-
ancestor order for every finalized identity, not merely the same identity set. -/
theorem effectiveRequired_first_identity (layers : List AliasContractLayer)
    (identity : Nat × Nat) :
    (effectiveRequired layers).find? (fun field => field.identity == identity) =
      ((layers.reverse.flatMap (·.required)).find?
        (fun field => field.identity == identity)) := by
  simpa [effectiveRequired] using foldRequired_first_identity layers.reverse [] identity

/-- Every independently extracted raw required identity occurs in the
candidate union. -/
theorem effectiveRequired_contains_raw (layers : List AliasContractLayer)
    (layer : AliasContractLayer) (field : RequiredField)
    (layerPresent : layer ∈ layers) (fieldPresent : field ∈ layer.required) :
    field.identity ∈ (effectiveRequired layers).map RequiredField.identity := by
  have reversed : layer ∈ layers.reverse := by simpa using layerPresent
  have flattened : field ∈ layers.reverse.flatMap (·.required) :=
    List.mem_flatMap.mpr ⟨layer, reversed, fieldPresent⟩
  by_cases present : field.identity ∈
      (effectiveRequired layers).map RequiredField.identity
  · exact present
  · exfalso
    have outputNone :
        (effectiveRequired layers).find?
            (fun candidate => candidate.identity == field.identity) = none := by
      apply List.find?_eq_none.mpr
      intro candidate member equal
      apply present
      exact List.mem_map.mpr ⟨candidate, member, beq_iff_eq.mp equal⟩
    have rawNotNone :
        (layers.reverse.flatMap (·.required)).find?
          (fun candidate => candidate.identity == field.identity) ≠ none := by
      intro absent
      have noMatch := List.find?_eq_none.mp absent
      exact (noMatch field flattened) (by simp)
    rw [effectiveRequired_first_identity] at outputNone
    exact rawNotNone outputNone

/-- The enum authored by the current declaration, if any. An inherited enum is
already valid at its authoring prefix and composes with later non-enum clauses
as an intersection; it is not a new value assertion at every descendant. -/
def currentAuthoredEnumeration (layers : List AliasContractLayer) :
    Option (List AuthoredContractValue) :=
  (layers.getLast?).bind (·.enumeration)

/-- Validate the enum authored at one raw ancestry prefix and the selected
default at that prefix. This prevents a later narrowing or local replacement
from hiding an invalid authored ancestor without applying later predicates
retroactively to inherited enum declarations. -/
def validateAliasPrefix (layers : List AliasContractLayer) :
    Except AliasContractError Unit :=
  match effectiveEnumeration layers with
  | .error error => .error error
  | .ok enumeration =>
      let defaultValue := selectedDefault layers
      let numericLayers := layers.map (·.numeric)
      let predicates := effectivePredicates layers
      match firstInvalidEffectiveEnum (currentAuthoredEnumeration layers)
          numericLayers predicates with
      | some value => .error (.invalidEnum value.origin value.semantic)
      | none =>
          match defaultValue with
          | none => .ok ()
          | some value =>
              if effectiveContractAllows enumeration numericLayers predicates value then
                .ok ()
              else .error (.invalidDefault value.origin value.semantic)

private def validateAliasPrefixesFrom (accumulated : List AliasContractLayer) :
    List AliasContractLayer → Except AliasContractError Unit
  | [] => .ok ()
  | layer :: rest =>
      let next := accumulated ++ [layer]
      match validateAliasPrefix next with
      | .error error => .error error
      | .ok _ => validateAliasPrefixesFrom next rest

def validateAliasPrefixes (layers : List AliasContractLayer) :
    Except AliasContractError Unit :=
  validateAliasPrefixesFrom [] layers

/-- Independent recursive shape of the prefix obligation: every authored enum
is checked at its declaration, and every selected default is checked under the
enum, numeric and symbolic predicate contract available at each prefix. -/
def EveryAliasPrefixAdmittedFrom (accumulated : List AliasContractLayer) :
    List AliasContractLayer → Prop
  | [] => True
  | layer :: rest =>
      let next := accumulated ++ [layer]
      validateAliasPrefix next = .ok () ∧ EveryAliasPrefixAdmittedFrom next rest

theorem validateAliasPrefixesFrom_iff (accumulated remaining) :
    validateAliasPrefixesFrom accumulated remaining = .ok () ↔
      EveryAliasPrefixAdmittedFrom accumulated remaining := by
  induction remaining generalizing accumulated with
  | nil => simp [validateAliasPrefixesFrom, EveryAliasPrefixAdmittedFrom]
  | cons layer rest ih =>
    simp only [validateAliasPrefixesFrom, EveryAliasPrefixAdmittedFrom]
    cases hprefix : validateAliasPrefix (accumulated ++ [layer]) with
    | error error => simp
    | ok value =>
      cases value
      simp [ih]

theorem validateAliasPrefixes_iff (layers : List AliasContractLayer) :
    validateAliasPrefixes layers = .ok () ↔
      EveryAliasPrefixAdmittedFrom [] layers := by
  exact validateAliasPrefixesFrom_iff [] layers

/-- Candidate evaluation consumes raw layers only. Every base-to-current prefix
is admitted before the final contract is returned. The result retains every
numeric layer, stably de-duplicates typed predicates and required identities,
and preserves their defined provenance precedence. -/
def evaluateAliasContract (layers : List AliasContractLayer) :
    Except AliasContractError EffectiveAliasContract :=
  match validateAliasPrefixes layers with
  | .error error => .error error
  | .ok _ =>
    match effectiveEnumeration layers with
    | .error error => .error error
    | .ok enumeration =>
      let defaultValue := selectedDefault layers
      let numericLayers := layers.map (·.numeric)
      let predicates := effectivePredicates layers
      let result : EffectiveAliasContract := {
        enumeration := enumeration
        defaultValue := defaultValue
        numericLayers := numericLayers
        predicates := predicates
        required := effectiveRequired layers
      }
      match firstInvalidEffectiveEnum (currentAuthoredEnumeration layers)
          numericLayers predicates with
      | some value => .error (.invalidEnum value.origin value.semantic)
      | none =>
          match defaultValue with
          | none => .ok result
          | some value =>
              if effectiveContractAllows enumeration numericLayers predicates value then
                .ok result
              else .error (.invalidDefault value.origin value.semantic)

def aliasOutcomeEqual (left right : Except AliasContractError EffectiveAliasContract) : Bool :=
  match left, right with
  | .ok a, .ok b => a == b
  | .error a, .error b => a == b
  | _, _ => false

theorem evaluated_prefixes_valid {layers effective}
    (success : evaluateAliasContract layers = .ok effective) :
    validateAliasPrefixes layers = .ok () := by
  unfold evaluateAliasContract at success
  cases hprefix : validateAliasPrefixes layers with
  | error error => simp [hprefix] at success
  | ok value =>
    cases value
    rfl

theorem evaluated_default_valid {layers effective}
    (success : evaluateAliasContract layers = .ok effective) :
    ∀ value ∈ effective.defaultValue,
      effectiveContractAllows effective.enumeration effective.numericLayers
        effective.predicates value = true := by
  have prefixes := evaluated_prefixes_valid success
  unfold evaluateAliasContract at success
  simp only [prefixes] at success
  cases henum : effectiveEnumeration layers with
  | error error => simp [henum] at success
  | ok enumeration =>
    simp only [henum] at success
    cases hinvalid : firstInvalidEffectiveEnum (currentAuthoredEnumeration layers)
        (layers.map (·.numeric)) (effectivePredicates layers) with
    | some invalid => simp [hinvalid] at success
    | none =>
      simp only [hinvalid] at success
      cases hdefault : selectedDefault layers with
      | none => simp [hdefault] at success; subst effective; simp
      | some default =>
        simp only [hdefault] at success
        split at success
        next accepted => cases success; simp_all
        next rejected => simp_all

/-- Every enum member authored at the final declaration of a successful
candidate satisfies the numeric, Pattern and Format clauses effective at that
declaration. Earlier authored enums have the same obligation at their own
prefix, as exposed by `evaluated_prefixes_valid`. -/
theorem evaluated_current_authored_enumeration_valid {layers effective}
    (success : evaluateAliasContract layers = .ok effective) :
    ∀ values ∈ currentAuthoredEnumeration layers, ∀ value ∈ values,
      nonEnumContractAllows (layers.map (·.numeric))
        (effectivePredicates layers) value = true := by
  have prefixes := evaluated_prefixes_valid success
  unfold evaluateAliasContract at success
  simp only [prefixes] at success
  cases henum : effectiveEnumeration layers with
  | error error => simp [henum] at success
  | ok enumeration =>
    simp only [henum] at success
    cases hinvalid : firstInvalidEffectiveEnum (currentAuthoredEnumeration layers)
        (layers.map (·.numeric)) (effectivePredicates layers) with
    | some invalid => simp [hinvalid] at success
    | none =>
      have valid := (firstInvalidEffectiveEnum_none_iff
        (currentAuthoredEnumeration layers)
        (layers.map (·.numeric)) (effectivePredicates layers)).1 hinvalid
      simp only [hinvalid] at success
      cases hdefault : selectedDefault layers with
      | none =>
        simp [hdefault] at success
        cases success
        simpa using valid
      | some default =>
        simp only [hdefault] at success
        split at success
        next accepted =>
          cases success
          simpa using valid
        next rejected => simp_all

theorem evaluated_numeric_layers_exact {layers effective}
    (success : evaluateAliasContract layers = .ok effective) :
    effective.numericLayers = layers.map (·.numeric) := by
  have prefixes := evaluated_prefixes_valid success
  unfold evaluateAliasContract at success
  simp only [prefixes] at success
  cases henum : effectiveEnumeration layers with
  | error error => simp [henum] at success
  | ok enumeration =>
    simp only [henum] at success
    cases hinvalid : firstInvalidEffectiveEnum (currentAuthoredEnumeration layers)
        (layers.map (·.numeric)) (effectivePredicates layers) with
    | some invalid => simp [hinvalid] at success
    | none =>
      simp only [hinvalid] at success
      cases hdefault : selectedDefault layers with
      | none => simp [hdefault] at success; subst effective; simp
      | some default =>
        simp only [hdefault] at success
        split at success
        next accepted => cases success; simp
        next rejected => simp_all

theorem evaluated_predicates_exact {layers effective}
    (success : evaluateAliasContract layers = .ok effective) :
    effective.predicates = effectivePredicates layers := by
  have prefixes := evaluated_prefixes_valid success
  unfold evaluateAliasContract at success
  simp only [prefixes] at success
  cases henum : effectiveEnumeration layers with
  | error error => simp [henum] at success
  | ok enumeration =>
    simp only [henum] at success
    cases hinvalid : firstInvalidEffectiveEnum (currentAuthoredEnumeration layers)
        (layers.map (·.numeric)) (effectivePredicates layers) with
    | some invalid => simp [hinvalid] at success
    | none =>
      simp only [hinvalid] at success
      cases hdefault : selectedDefault layers with
      | none => simp [hdefault] at success; subst effective; simp
      | some default =>
        simp only [hdefault] at success
        split at success
        next accepted => cases success; simp
        next rejected => simp_all

theorem evaluated_required_exact {layers effective}
    (success : evaluateAliasContract layers = .ok effective) :
    effective.required = effectiveRequired layers := by
  have prefixes := evaluated_prefixes_valid success
  unfold evaluateAliasContract at success
  simp only [prefixes] at success
  cases henum : effectiveEnumeration layers with
  | error error => simp [henum] at success
  | ok enumeration =>
    simp only [henum] at success
    cases hinvalid : firstInvalidEffectiveEnum (currentAuthoredEnumeration layers)
        (layers.map (·.numeric)) (effectivePredicates layers) with
    | some invalid => simp [hinvalid] at success
    | none =>
      simp only [hinvalid] at success
      cases hdefault : selectedDefault layers with
      | none => simp [hdefault] at success; subst effective; simp
      | some default =>
        simp only [hdefault] at success
        split at success
        next accepted => cases success; simp
        next rejected => simp_all

theorem numeric_layers_conjunction (layers : List AuthoredNumericBounds)
    (value : AuthoredContractValue) :
    numericLayersAllow layers value = true ↔
      ∀ number ∈ value.number, ∀ bounds ∈ layers,
        bounds.Allows number := by
  simp only [numericLayersAllow, Option.all_eq_true, List.all_eq_true]
  constructor <;> intro accepted number present bounds member
  · exact (effectiveNumericBounds_iff bounds number).1 (accepted number present bounds member)
  · exact (effectiveNumericBounds_iff bounds number).2 (accepted number present bounds member)

theorem predicate_clauses_conjunction (clauses : List AuthoredPredicateClause)
    (value : AuthoredContractValue) :
    predicateClausesAllow clauses value = true ↔
    ∀ clause ∈ clauses, clause.identity ∈ value.satisfiedPredicates := by
  simp [predicateClausesAllow]

/-- Successful equal and proper-subset declarations, with a replacing local
default, demonstrate that refinement is not vacuous. -/
theorem enum_refinement_positive_controls :
    let one : AuthoredContractValue := { origin := 10, semantic := 1 }
    let two : AuthoredContractValue := { origin := 11, semantic := 2 }
    let inherited : AliasContractLayer :=
      { declaration := 100, enumeration := some [one, two], defaultValue := some one }
    let equal : AliasContractLayer :=
      { declaration := 101, enumeration := some [one, two] }
    let subset : AliasContractLayer :=
      { declaration := 102, enumeration := some [two], defaultValue := some two }
    (evaluateAliasContract [inherited, equal]).isOk = true ∧
      aliasOutcomeEqual (evaluateAliasContract [inherited, subset]) (.ok
        { enumeration := some [two], defaultValue := some two,
          numericLayers := [{}, {}], predicates := [], required := [] }) = true := by decide

/-- Partial overlap and disjoint replacement report the first authored member
outside the effective ancestor enum. -/
theorem enum_refinement_rejection_controls :
    let one : AuthoredContractValue := { origin := 10, semantic := 1 }
    let two : AuthoredContractValue := { origin := 11, semantic := 2 }
    let three : AuthoredContractValue := { origin := 12, semantic := 3 }
    let parent : AliasContractLayer :=
      { declaration := 100, enumeration := some [one, two] }
    let overlap : AliasContractLayer :=
      { declaration := 101, enumeration := some [two, three] }
    let disjoint : AliasContractLayer :=
      { declaration := 102, enumeration := some [three] }
    aliasOutcomeEqual (evaluateAliasContract [parent, overlap])
        (.error (.enumWidening 1 101 12 3)) = true ∧
      aliasOutcomeEqual (evaluateAliasContract [parent, disjoint])
        (.error (.enumWidening 1 102 12 3)) = true := by decide

/-- The legacy last-declaration rule admits the partial-overlap defect. -/
theorem legacy_enum_override_counterexample :
    let one : AuthoredContractValue := { origin := 10, semantic := 1 }
    let two : AuthoredContractValue := { origin := 11, semantic := 2 }
    let three : AuthoredContractValue := { origin := 12, semantic := 3 }
    let layers : List AliasContractLayer := [
      { declaration := 100, enumeration := some [one, two] },
      { declaration := 101, enumeration := some [two, three] }]
    (legacyEffectiveEnum layers).any (fun values => enumContains values three) = true ∧
      aliasOutcomeEqual (evaluateAliasContract layers)
        (.error (.enumWidening 1 101 12 3)) = true := by decide

/-- A narrowed enum invalidates an inherited default, while a valid explicit
local default replaces it. Absent remains distinct from a present default. -/
theorem effective_default_controls :
    let one : AuthoredContractValue := { origin := 10, semantic := 1 }
    let two : AuthoredContractValue := { origin := 11, semantic := 2 }
    let parent : AliasContractLayer :=
      { declaration := 100, enumeration := some [one, two], defaultValue := some one }
    let narrowed : AliasContractLayer :=
      { declaration := 101, enumeration := some [two] }
    let replaced : AliasContractLayer :=
      { declaration := 102, enumeration := some [two], defaultValue := some two }
    aliasOutcomeEqual (evaluateAliasContract [parent, narrowed])
        (.error (.invalidDefault 10 1)) = true ∧
      (evaluateAliasContract [parent, replaced]).isOk = true ∧
      selectedDefault [{ declaration := 103 }] = none := by decide

/-- Required fields accumulate by finalized identity and retain the first
provenance in effective-to-ancestor precedence order. -/
theorem required_union_controls :
    let first : RequiredField := { declaration := 5, field := 7, origin := 10 }
    let duplicate : RequiredField := { declaration := 5, field := 7, origin := 11 }
    let second : RequiredField := { declaration := 5, field := 8, origin := 12 }
    effectiveRequired [
      { declaration := 100, required := [first] },
      { declaration := 101, required := [duplicate, second] }] = [duplicate, second] := by decide

/-- Four bounds remain independent at every alias layer; equal-value open ties
and contradictory intervals are accepted contracts, even when they admit no value. -/
theorem alias_numeric_controls :
    let layers : List AuthoredNumericBounds := [
      { minimum := some ⟨1, 0⟩, exclusiveMinimum := some ⟨10, -1⟩ },
      { maximum := some ⟨5, 0⟩, exclusiveMaximum := some ⟨50, -1⟩ }]
    let empty : List AuthoredNumericBounds := [
      { minimum := some ⟨5, 0⟩ }, { maximum := some ⟨1, 0⟩ }]
    numericLayersAllow layers { origin := 1, semantic := 1, number := some ⟨1, 0⟩ } = false ∧
      numericLayersAllow layers { origin := 2, semantic := 2, number := some ⟨2, 0⟩ } = true ∧
      numericLayersAllow empty { origin := 3, semantic := 3, number := some ⟨3, 0⟩ } = false ∧
      (evaluateAliasContract [
        { declaration := 1, numeric := { minimum := some ⟨5, 0⟩ } },
        { declaration := 2, numeric := { maximum := some ⟨1, 0⟩ } }]).isOk = true := by decide

def legacyNearestPredicate (kind : PredicateKind) (layers : List AliasContractLayer) :
    Option AuthoredPredicateClause :=
  layers.foldl (fun current layer =>
    (layer.predicates.find? fun clause => clause.kind == kind).or current) none

def legacyNearestPredicatesAllow (layers : List AliasContractLayer)
    (value : AuthoredContractValue) : Bool :=
  [.pattern, .format].all fun kind =>
    (legacyNearestPredicate kind layers).all fun clause =>
      value.satisfiedPredicates.contains clause.identity

/-- Pattern and Format IDs occupy disjoint identity domains. Clauses traverse
current-to-base, exact duplicates retain current provenance, and distinct
clauses conjoin. -/
theorem predicate_clause_union_controls :
    let pattern : PredicateIdentity := { kind := .pattern, predicate := 7 }
    let format : PredicateIdentity := { kind := .format, predicate := 7 }
    let basePattern : AuthoredPredicateClause :=
      { kind := .pattern, predicate := 7, origin := 10 }
    let currentPattern : AuthoredPredicateClause :=
      { kind := .pattern, predicate := 7, origin := 20 }
    let currentFormat : AuthoredPredicateClause :=
      { kind := .format, predicate := 7, origin := 30 }
    let value : AuthoredContractValue :=
      { origin := 40, semantic := 1, satisfiedPredicates := [pattern, format] }
    let layers : List AliasContractLayer := [
      { declaration := 100, predicates := [basePattern] },
      { declaration := 101, predicates := [currentPattern, currentFormat],
        defaultValue := some value }]
    pattern ≠ format ∧
      effectivePredicates layers = [currentPattern, currentFormat] ∧
      predicateClausesAllow (effectivePredicates layers) value = true ∧
      (evaluateAliasContract layers).isOk = true := by decide

/-- The legacy nearest-Pattern rule admits a default that passes the current
Pattern but violates an independently authored base Pattern. -/
theorem legacy_nearest_pattern_counterexample :
    let base : AuthoredPredicateClause :=
      { kind := .pattern, predicate := 1, origin := 10 }
    let current : AuthoredPredicateClause :=
      { kind := .pattern, predicate := 2, origin := 20 }
    let value : AuthoredContractValue := {
      origin := 30, semantic := 1,
      satisfiedPredicates := [{ kind := .pattern, predicate := 2 }]
    }
    let layers : List AliasContractLayer := [
      { declaration := 100, predicates := [base] },
      { declaration := 101, predicates := [current], defaultValue := some value }]
    legacyNearestPredicatesAllow layers value = true ∧
      aliasOutcomeEqual (evaluateAliasContract layers)
        (.error (.invalidDefault 30 1)) = true := by decide

/-- Pattern IDs cannot satisfy Format clauses. The legacy nearest-Format rule
therefore also admits a current-valid default that violates the base Format. -/
theorem legacy_nearest_format_counterexample :
    let base : AuthoredPredicateClause :=
      { kind := .format, predicate := 1, origin := 10 }
    let current : AuthoredPredicateClause :=
      { kind := .format, predicate := 2, origin := 20 }
    let value : AuthoredContractValue := {
      origin := 30, semantic := 1,
      satisfiedPredicates := [
        { kind := .pattern, predicate := 1 },
        { kind := .format, predicate := 2 }]
    }
    let layers : List AliasContractLayer := [
      { declaration := 100, predicates := [base] },
      { declaration := 101, predicates := [current], defaultValue := some value }]
    legacyNearestPredicatesAllow layers value = true ∧
      predicateClausesAllow (effectivePredicates layers) value = false ∧
      aliasOutcomeEqual (evaluateAliasContract layers)
        (.error (.invalidDefault 30 1)) = true := by decide

/-- A valid ancestor declaration is checked under its own contract. A later
predicate may exclude an ancestor enum member or default when the descendant
validly narrows and replaces them. -/
theorem prefix_validation_nonretroactive_controls :
    let basePattern : PredicateIdentity := { kind := .pattern, predicate := 1 }
    let currentPattern : PredicateIdentity := { kind := .pattern, predicate := 2 }
    let ancestorOnly : AuthoredContractValue :=
      { origin := 10, semantic := 1, satisfiedPredicates := [basePattern] }
    let current : AuthoredContractValue :=
      { origin := 11, semantic := 2,
        satisfiedPredicates := [basePattern, currentPattern] }
    let layers : List AliasContractLayer := [
      { declaration := 100, enumeration := some [ancestorOnly, current],
        defaultValue := some ancestorOnly,
        predicates := [{ kind := .pattern, predicate := 1, origin := 100 }] },
      { declaration := 101, enumeration := some [current], defaultValue := some current,
        predicates := [{ kind := .pattern, predicate := 2, origin := 101 }] }]
    (evaluateAliasContract layers).isOk = true ∧
      (validateAliasPrefixes layers).isOk = true := by decide

/-- A descendant narrowing cannot hide an enum member that was invalid when
authored at its ancestor declaration. -/
theorem invalid_superseded_ancestor_enum_control :
    let basePattern : PredicateIdentity := { kind := .pattern, predicate := 1 }
    let invalid : AuthoredContractValue := { origin := 10, semantic := 1 }
    let valid : AuthoredContractValue :=
      { origin := 11, semantic := 2, satisfiedPredicates := [basePattern] }
    let layers : List AliasContractLayer := [
      { declaration := 100, enumeration := some [invalid, valid],
        predicates := [{ kind := .pattern, predicate := 1, origin := 100 }] },
      { declaration := 101, enumeration := some [valid] }]
    aliasOutcomeEqual (evaluateAliasContract layers)
      (.error (.invalidEnum 10 1)) = true := by decide

/-- Inherited enum members compose with later predicates as an intersection:
the declaration remains valid while runtime admission accepts only members
that satisfy the full contract. Authoring that same enum locally is a new value
assertion and is rejected. Selected defaults remain value assertions at every
prefix, so an excluded inherited default must be replaced locally. -/
theorem inherited_enum_conjunction_and_local_authorship_controls :
    let basePattern : PredicateIdentity := { kind := .pattern, predicate := 1 }
    let derivedPattern : PredicateIdentity := { kind := .pattern, predicate := 2 }
    let retained : AuthoredContractValue :=
      { origin := 10, semantic := 1,
        satisfiedPredicates := [basePattern, derivedPattern] }
    let excluded : AuthoredContractValue :=
      { origin := 11, semantic := 2, satisfiedPredicates := [basePattern] }
    let inheritedLayers : List AliasContractLayer := [
      { declaration := 100, enumeration := some [retained, excluded],
        predicates := [{ kind := .pattern, predicate := 1, origin := 100 }] },
      { declaration := 101,
        predicates := [{ kind := .pattern, predicate := 2, origin := 101 }] }]
    let localLayers : List AliasContractLayer := [
      { declaration := 100, enumeration := some [retained, excluded],
        predicates := [{ kind := .pattern, predicate := 1, origin := 100 }] },
      { declaration := 101, enumeration := some [retained, excluded],
        predicates := [{ kind := .pattern, predicate := 2, origin := 101 }] }]
    let inheritedDefaultLayers : List AliasContractLayer := [
      { declaration := 100, enumeration := some [retained, excluded],
        defaultValue := some excluded,
        predicates := [{ kind := .pattern, predicate := 1, origin := 100 }] },
      { declaration := 101,
        predicates := [{ kind := .pattern, predicate := 2, origin := 101 }] }]
    let replacedDefaultLayers : List AliasContractLayer := [
      { declaration := 100, enumeration := some [retained, excluded],
        defaultValue := some excluded,
        predicates := [{ kind := .pattern, predicate := 1, origin := 100 }] },
      { declaration := 101, defaultValue := some retained,
        predicates := [{ kind := .pattern, predicate := 2, origin := 101 }] }]
    (evaluateAliasContract inheritedLayers).isOk = true ∧
      effectiveContractAllows (some [retained, excluded])
        (inheritedLayers.map (·.numeric))
        (effectivePredicates inheritedLayers) retained = true ∧
      effectiveContractAllows (some [retained, excluded])
        (inheritedLayers.map (·.numeric))
        (effectivePredicates inheritedLayers) excluded = false ∧
      aliasOutcomeEqual (evaluateAliasContract localLayers)
        (.error (.invalidEnum 11 2)) = true ∧
      aliasOutcomeEqual (evaluateAliasContract inheritedDefaultLayers)
        (.error (.invalidDefault 11 2)) = true ∧
      (evaluateAliasContract replacedDefaultLayers).isOk = true := by decide

/-- A valid local replacement cannot hide a default that was invalid at its
ancestor declaration. -/
theorem invalid_superseded_ancestor_default_control :
    let basePattern : PredicateIdentity := { kind := .pattern, predicate := 1 }
    let invalid : AuthoredContractValue := { origin := 10, semantic := 1 }
    let valid : AuthoredContractValue :=
      { origin := 11, semantic := 2, satisfiedPredicates := [basePattern] }
    let layers : List AliasContractLayer := [
      { declaration := 100, defaultValue := some invalid,
        predicates := [{ kind := .pattern, predicate := 1, origin := 100 }] },
      { declaration := 101, defaultValue := some valid }]
    aliasOutcomeEqual (evaluateAliasContract layers)
      (.error (.invalidDefault 10 1)) = true := by decide

/-- Every intermediate prefix is checked. A final replacement cannot hide an
inherited default that became invalid at a middle declaration. -/
theorem invalid_middle_inherited_default_control :
    let basePattern : PredicateIdentity := { kind := .pattern, predicate := 1 }
    let middlePattern : PredicateIdentity := { kind := .pattern, predicate := 2 }
    let inherited : AuthoredContractValue :=
      { origin := 10, semantic := 1, satisfiedPredicates := [basePattern] }
    let replacement : AuthoredContractValue :=
      { origin := 11, semantic := 2,
        satisfiedPredicates := [basePattern, middlePattern] }
    let layers : List AliasContractLayer := [
      { declaration := 100, defaultValue := some inherited,
        predicates := [{ kind := .pattern, predicate := 1, origin := 100 }] },
      { declaration := 101,
        predicates := [{ kind := .pattern, predicate := 2, origin := 101 }] },
      { declaration := 102, defaultValue := some replacement }]
    aliasOutcomeEqual (evaluateAliasContract layers)
      (.error (.invalidDefault 10 1)) = true := by decide

end ValueContract.Candidate
