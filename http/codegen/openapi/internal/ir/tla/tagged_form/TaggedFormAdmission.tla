------------------------- MODULE TaggedFormAdmission ------------------------
EXTENDS Naturals
CONSTANT RejectNull
VARIABLE source
vars == <<source>>
Init == source \in {"empty", "field", "null", "tag-collision"}
Next == UNCHANGED vars
Admitted == source # "tag-collision" /\ (~RejectNull \/ source # "null")
Wire == IF source = "field" THEN {"tag", "field"} ELSE {"tag"}
Decoded == IF "field" \in Wire THEN "field" ELSE "empty"
Representable == Admitted => Decoded = source
Spec == Init /\ [][Next]_vars
=============================================================================
