------------------------- MODULE ExplicitNames -------------------------
EXTENDS Naturals, FiniteSets
CONSTANT PreserveNames
Positions == {"root", "field", "element", "mapValue", "unionBranch", "wrapper"}
VARIABLE normalized, references, declarations
vars == <<normalized, references, declarations>>
Init == /\ normalized = {}
        /\ references = [p \in Positions |-> "unset"]
        /\ declarations = {}
Normalize(p) ==
  LET name == IF PreserveNames \/ p = "root" THEN "Authored" ELSE "Type" IN
  /\ p \notin normalized
  /\ normalized' = normalized \cup {p}
  /\ references' = [references EXCEPT ![p] = name]
  /\ declarations' = declarations \cup {name}
Done == normalized = Positions
Next == (\E p \in Positions: Normalize(p)) \/ (Done /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
AuthoredIntent == \A p \in normalized: references[p] = "Authored"
ReferencesDeclared == \A p \in normalized: references[p] \in declarations
OneIdentity == Cardinality(declarations) <= 1
Terminates == <>Done
=============================================================================
