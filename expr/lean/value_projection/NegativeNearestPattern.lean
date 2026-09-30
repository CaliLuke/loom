import ValueContract.AliasContracts

open ValueContract.Candidate

-- Deliberately false: a nearest-only Pattern check is not the authored
-- current-to-base conjunction.
example :
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
      { declaration := 101, predicates := [current] }]
    legacyNearestPredicatesAllow layers value =
      predicateClausesAllow (effectivePredicates layers) value := by
  decide
