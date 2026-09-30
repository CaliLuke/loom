import ValueContract.AliasContracts

open ValueContract.Candidate

-- Deliberately false: the legacy last declaration is not a valid refinement.
example :
    let one : AuthoredContractValue := { origin := 1, semantic := 1 }
    let two : AuthoredContractValue := { origin := 2, semantic := 2 }
    let three : AuthoredContractValue := { origin := 3, semantic := 3 }
    let layers : List AliasContractLayer := [
      { declaration := 10, enumeration := some [one, two] },
      { declaration := 11, enumeration := some [two, three] }]
    (effectiveEnumeration layers).isOk = true := by
  decide
