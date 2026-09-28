----------------------------- MODULE JSONOptions -----------------------------
EXTENDS Naturals, Sequences, FiniteSets
CONSTANTS Mode, MaxDepth
VARIABLES chain, direction, position, option, outcome
vars == <<chain, direction, position, option, outcome>>
Layers == {"union", "optional", "nullable"}
Chains == UNION {[1..n -> Layers]: n \in 1..MaxDepth}

Init == /\ chain \in Chains
        /\ direction \in {"encode", "decode"}
        /\ position = 1
        /\ option = TRUE
        /\ outcome = "pending"

RestartJSON ==
  /\ outcome = "pending"
  /\ position <= Len(chain)
  /\ option' = (Mode = "all" \/ (Mode = "union" /\ chain[position] = "union"))
  /\ position' = position + 1
  /\ UNCHANGED <<chain, direction, outcome>>

MapCodec ==
  /\ outcome = "pending"
  /\ position > Len(chain)
  /\ outcome' = IF option THEN "success" ELSE "rejected"
  /\ UNCHANGED <<chain, direction, position, option>>

Next == RestartJSON \/ MapCodec
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
CanonicalMapsWork == outcome # "rejected"
EventuallyFinished == <> (outcome # "pending")
=============================================================================
