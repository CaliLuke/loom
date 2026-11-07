------------------------ MODULE FormUnionPresence ------------------------
EXTENDS FiniteSets
CONSTANT Policy
VARIABLES wire, optional, hasInput, schema, decoded, accepted, phase
vars == <<wire, optional, hasInput, schema, decoded, accepted, phase>>
Init == /\ wire \in {"empty", "zero", "nonzero", "invalid"}
        /\ optional \in BOOLEAN
        /\ hasInput \in BOOLEAN
        /\ (wire # "empty" => hasInput)
        /\ schema = {} /\ decoded = {} /\ accepted = FALSE
        /\ phase = "count"
Count == /\ phase = "count"
         /\ schema' = IF wire = "empty" THEN {"Empty"}
                       ELSE IF wire # "invalid" THEN {"Count"} ELSE {}
         /\ decoded' = (IF wire = "empty" THEN {"Empty"} ELSE {}) \cup
              (IF wire # "invalid" /\ (Policy = "native" \/ wire # "empty")
               THEN {"Count"} ELSE {})
         /\ phase' = "select"
         /\ UNCHANGED <<wire, optional, hasInput, accepted>>
Select == /\ phase = "select"
          /\ accepted' = ((optional /\ ~hasInput) \/
                         (Cardinality(schema) = 1 /\ schema = decoded))
          /\ phase' = "done"
          /\ UNCHANGED <<wire, optional, hasInput, schema, decoded>>
Next == Count \/ Select \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
BodyAndMemberPresence == phase = "done" => accepted = (wire # "invalid")
ZeroIsSupplied == phase = "done" /\ wire = "zero" => decoded = {"Count"}
=============================================================================
