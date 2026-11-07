-------------------------- MODULE NestedText --------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Strategy
VARIABLES schema, decoder, phase, result
vars == <<schema, decoder, phase, result>>
Leaves == {1, 2, 3}
Inner == {1, 2}
Unique(s) == Cardinality(s) = 1
Agrees(s, d) == Unique(s) /\ s = d
Init == /\ schema \in SUBSET Leaves /\ decoder \in SUBSET Leaves
        /\ phase = "input" /\ result = FALSE
InnerSchema == Unique(schema \cap Inner)
InnerDecode == Agrees(schema \cap Inner, decoder \cap Inner)
OuterSchema == (IF InnerSchema THEN {1} ELSE {}) \cup (IF 3 \in schema THEN {2} ELSE {})
OuterDecode == (IF InnerDecode THEN {1} ELSE {}) \cup (IF 3 \in decoder THEN {2} ELSE {})
Expected == Agrees(OuterSchema, OuterDecode)
Select == /\ phase = "input" /\ phase' = "selected"
          /\ result' = IF Strategy = "flatten" THEN Agrees(schema, decoder) ELSE Expected
          /\ UNCHANGED <<schema, decoder>>
Next == Select
Spec == Init /\ [][Next]_vars
Agreement == phase = "selected" => result = Expected
=======================================================================
