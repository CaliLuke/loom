import Std

namespace Loom.OpenAPI.DocumentIdentity

inductive PathSegment where
  | literal (identity : Nat)
  | parameter
  deriving DecidableEq, Repr

structure OperationClaim where
  source : Nat
  pathKey : Nat
  template : List PathSegment
  method : Nat
  operationId : Option Nat
  deriving DecidableEq, Repr

structure ComponentClaim where
  name : Option Nat
  value : Nat
  deriving DecidableEq, Repr

def uniqueBy {α β : Type} [DecidableEq β] (project : α → β) (items : List α) : Bool :=
  (items.map project).Nodup

def uniqueTemplates (items : List OperationClaim) : Bool :=
  items.all fun left =>
    items.all fun right =>
      left.pathKey == right.pathKey || left.template != right.template

def uniqueRouteSlots (items : List OperationClaim) : Bool :=
  uniqueBy (fun claim => (claim.pathKey, claim.method)) items

def uniqueOperationIDs (items : List OperationClaim) : Bool :=
  uniqueBy id (items.filterMap OperationClaim.operationId)

def consistentComponentClaims (items : List ComponentClaim) : Bool :=
  items.all fun left =>
    items.all fun right =>
      left.name != right.name || left.value == right.value

def accepted (operations : List OperationClaim) (components : List ComponentClaim) : Bool :=
  uniqueOperationIDs operations && uniqueTemplates operations &&
    uniqueRouteSlots operations && consistentComponentClaims components

-- The previous generators merged operations into raw path/method maps and
-- assigned named media components without document-level validation.
def legacyAccepts (_operations : List OperationClaim) (_components : List ComponentClaim) : Bool :=
  true

def legacyMediaComponent (items : List ComponentClaim) (name : Nat) : Option Nat :=
  items.foldl (fun current claim =>
    if claim.name == some name then some claim.value else current) none

def operationsWithEquivalentTemplates : List OperationClaim :=
  [ ⟨1, 1, [.literal 1, .parameter], 1, some 1⟩,
    ⟨2, 2, [.literal 1, .parameter], 2, some 2⟩ ]

def operationsWithEquivalentStaticSuffixTemplates : List OperationClaim :=
  [ ⟨1, 1, [.literal 1, .parameter, .literal 2], 1, some 1⟩,
    ⟨2, 2, [.literal 1, .parameter, .literal 2], 2, some 2⟩ ]

def operationsWithDistinctStaticSuffixTemplates : List OperationClaim :=
  [ ⟨1, 1, [.literal 1, .parameter, .literal 2], 1, some 1⟩,
    ⟨2, 2, [.literal 1, .parameter, .literal 3], 2, some 2⟩ ]

def operationsWithDuplicateIDs : List OperationClaim :=
  [ ⟨1, 1, [.literal 1], 1, some 7⟩,
    ⟨2, 2, [.literal 2], 1, some 7⟩ ]

def validOperationsAtOnePath : List OperationClaim :=
  [ ⟨1, 1, [.literal 1, .parameter], 1, some 1⟩,
    ⟨2, 1, [.literal 1, .parameter], 2, some 2⟩ ]

def duplicateRouteSlots : List OperationClaim :=
  [ ⟨1, 1, [.literal 1, .parameter], 1, some 1⟩,
    ⟨2, 1, [.literal 1, .parameter], 1, some 2⟩ ]

-- One authored route at route index zero expands through two service bases.
-- The old projection reuses the authored route index for both emitted paths.
def multiBaseWithAuthoredRouteIndex : List OperationClaim :=
  [ ⟨1, 1, [.literal 1, .parameter], 1, some 0⟩,
    ⟨1, 2, [.literal 2, .parameter], 1, some 0⟩ ]

-- The corrected projection keeps the first operation ID stable and uses the
-- emitted-path ordinal for the second path.
def multiBaseWithEmittedPathIndex : List OperationClaim :=
  [ ⟨1, 1, [.literal 1, .parameter], 1, some 0⟩,
    ⟨1, 2, [.literal 2, .parameter], 1, some 1⟩ ]

def conflictingMediaClaims : List ComponentClaim :=
  [ ⟨some 1, 1⟩, ⟨some 1, 2⟩ ]

def identicalMediaClaims : List ComponentClaim :=
  [ ⟨some 1, 1⟩, ⟨some 1, 1⟩ ]

theorem equivalent_templates_are_rejected :
    accepted operationsWithEquivalentTemplates [] = false := by
  decide

theorem equivalent_static_suffix_templates_are_rejected :
    accepted operationsWithEquivalentStaticSuffixTemplates [] = false := by
  decide

theorem distinct_static_suffix_templates_are_accepted :
    accepted operationsWithDistinctStaticSuffixTemplates [] = true := by
  decide

theorem baseline_accepts_equivalent_templates :
    legacyAccepts operationsWithEquivalentTemplates [] = true := by
  decide

theorem baseline_accepts_duplicate_operation_ids :
    legacyAccepts operationsWithDuplicateIDs [] = true := by
  decide

theorem baseline_accepts_conflicting_media_claims :
    legacyAccepts [] conflictingMediaClaims = true := by
  decide

theorem duplicate_operation_ids_are_rejected :
    accepted operationsWithDuplicateIDs [] = false := by
  decide

theorem path_methods_can_share_one_path_item :
    accepted validOperationsAtOnePath [] = true := by
  decide

theorem duplicate_route_slots_are_rejected :
    accepted duplicateRouteSlots [] = false := by
  decide

theorem multi_base_fanout_reusing_authored_route_index_is_rejected :
    accepted multiBaseWithAuthoredRouteIndex [] = false := by
  decide

theorem multi_base_fanout_uses_unique_emitted_path_indexes :
    accepted multiBaseWithEmittedPathIndex [] = true := by
  decide

theorem conflicting_explicit_media_names_are_rejected :
    accepted [] conflictingMediaClaims = false := by
  decide

theorem identical_explicit_media_names_can_share :
    accepted [] identicalMediaClaims = true := by
  decide

theorem legacy_media_claims_depend_on_order :
    legacyMediaComponent conflictingMediaClaims 1 !=
      legacyMediaComponent conflictingMediaClaims.reverse 1 := by
  decide

#print axioms equivalent_templates_are_rejected
#print axioms equivalent_static_suffix_templates_are_rejected
#print axioms distinct_static_suffix_templates_are_accepted
#print axioms baseline_accepts_equivalent_templates
#print axioms baseline_accepts_duplicate_operation_ids
#print axioms baseline_accepts_conflicting_media_claims
#print axioms duplicate_operation_ids_are_rejected
#print axioms path_methods_can_share_one_path_item
#print axioms duplicate_route_slots_are_rejected
#print axioms multi_base_fanout_reusing_authored_route_index_is_rejected
#print axioms multi_base_fanout_uses_unique_emitted_path_indexes
#print axioms conflicting_explicit_media_names_are_rejected
#print axioms identical_explicit_media_names_can_share
#print axioms legacy_media_claims_depend_on_order

end Loom.OpenAPI.DocumentIdentity
