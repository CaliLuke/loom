import ValueContract.Legacy
import Lean.Util.CollectAxioms
import Lean.Elab.Command

/- The manifest is reviewed source. This command inspects declaration bodies,
including imported dependencies; searching source text for `sorry` is insufficient.
The checker separately replays declarations in a fresh kernel environment. -/
open Lean Elab Command in
run_cmd do
  let manifest ← liftIO <| IO.FS.readFile "required-theorems.txt"
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
    logInfo m!"VALUE_CONTRACT_THEOREM {name} axioms={sorted}"
  logInfo m!"VALUE_CONTRACT_AUDIT_OK {names.length}"
