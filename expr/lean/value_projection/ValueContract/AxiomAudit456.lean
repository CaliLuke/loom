import ValueContract.Issue456CollisionDiagnostics
import Lean.Util.CollectAxioms
import Lean.Elab.Command

open Lean Elab Command in
run_cmd do
  let manifest ← liftIO <| IO.FS.readFile "required-theorems-456.txt"
  let names := (manifest.splitOn "\n").filter (· != "") |>.map String.toName
  if names.isEmpty then
    throwError "required theorem manifest is empty"
  if names.eraseDups.length != names.length then
    throwError "required theorem manifest contains duplicate names"
  let allowed := [``propext, ``Classical.choice, ``Quot.sound]
  for name in names do
    let declaration ← getConstInfo name
    unless declaration matches .thmInfo _ do
      throwError "required declaration {name} is not a theorem"
    let axioms ← collectAxioms name
    for axiomName in axioms do
      unless allowed.contains axiomName do
        throwError "forbidden transitive axiom {axiomName} in {name}"
    let sorted := axioms.toList.map Name.toString |>.mergeSort (· ≤ ·)
    logInfo m!"ISSUE456_THEOREM {name} axioms={sorted}"
  logInfo m!"ISSUE456_AUDIT_OK {names.length}"
