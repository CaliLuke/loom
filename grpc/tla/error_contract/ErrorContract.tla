---------------------------- MODULE ErrorContract ----------------------------
EXTENDS TLC
CONSTANT Aggregate
VARIABLES left, right, outer, code, details, phase
vars == <<left, right, outer, code, details, phase>>
Codes == {"unknown", "canceled", "unavailable"}
Consensus == IF left = right THEN left ELSE "unknown"

Init ==
  /\ left \in Codes
  /\ right \in Codes
  /\ outer \in BOOLEAN
  /\ code = "pending"
  /\ details = {}
  /\ phase = "classify"

Classify ==
  /\ phase = "classify"
  /\ code' = IF outer THEN "denied" ELSE IF Aggregate THEN Consensus ELSE left
  /\ details' = IF outer THEN {"outer"}
                  ELSE IF Aggregate THEN {"left", "right"} ELSE {"left"}
  /\ phase' = "done"
  /\ UNCHANGED <<left, right, outer>>

Spec == Init /\ [][Classify]_vars
CompleteFailure == phase = "done" /\ ~outer => details = {"left", "right"}
OrderIndependentStatus == phase = "done" /\ ~outer => code = Consensus
ExplicitOwner == phase = "done" /\ outer => code = "denied" /\ details = {"outer"}
=============================================================================
