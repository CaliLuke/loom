------------------------ MODULE JSONFieldPolicy ------------------------
EXTENDS Naturals
CONSTANT Policy
VARIABLES codec, nullable, required, value, accepted, phase
vars == <<codec, nullable, required, value, accepted, phase>>
Admitted == codec = "json" \/ ~nullable
Source == Admitted /\ (IF value = "absent" THEN ~required
                       ELSE value # "null" \/ nullable)
Init == /\ codec \in {"json", "text"}
        /\ nullable \in BOOLEAN /\ required \in BOOLEAN
        /\ value \in {"absent", "null", "value"}
        /\ accepted = FALSE /\ phase = "decode"
Decode == /\ phase = "decode"
          /\ accepted' = ((IF Policy = "blanket" THEN ~nullable ELSE Admitted)
                          /\ (IF value = "absent" THEN ~required
                              ELSE value # "null" \/ (Policy = "captured" /\ nullable)))
          /\ phase' = "done"
          /\ UNCHANGED <<codec, nullable, required, value>>
Next == Decode \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
Agreement == phase = "done" => accepted = Source
=============================================================================
