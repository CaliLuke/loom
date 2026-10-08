----------------------- MODULE ElementDecode -----------------------
EXTENDS TLC
CONSTANT Mode
VARIABLES shape, wire, nullable, selected, required, known, phase, rejected, converted, accepted
vars == <<shape, wire, nullable, selected, required, known, phase, rejected, converted, accepted>>
Init == /\ shape \in {"array", "map"}
        /\ wire \in {"absent", "empty", "value", "null-element"}
        /\ nullable \in BOOLEAN /\ selected \in BOOLEAN
        /\ required \in BOOLEAN /\ known \in BOOLEAN
        /\ phase = "wire" /\ rejected = FALSE
        /\ converted = "none" /\ accepted = FALSE
Decode == /\ phase = "wire"
          /\ rejected' = (Mode = "strict" /\ ~nullable /\ wire = "null-element")
          /\ phase' = "decoded"
          /\ UNCHANGED <<shape, wire, nullable, selected, required, known, converted, accepted>>
Convert == /\ phase = "decoded"
           /\ converted' = IF rejected THEN "none"
                           ELSE IF ~nullable /\ wire = "null-element"
                           THEN IF shape = "array" THEN "zero-element" ELSE "empty"
                           ELSE wire
           /\ phase' = "converted"
           /\ UNCHANGED <<shape, wire, nullable, selected, required, known, rejected, accepted>>
Validate == /\ phase = "converted"
            /\ accepted' = (~rejected /\ known /\ (~selected \/ ~required \/ converted # "absent"))
            /\ phase' = "done"
            /\ UNCHANGED <<shape, wire, nullable, selected, required, known, rejected, converted>>
Next == Decode \/ Convert \/ Validate \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
NoErasure == phase = "done" /\ accepted => converted = wire
Contract == phase = "done" => accepted =
 (known /\ (~selected \/ ~required \/ wire # "absent") /\ (nullable \/ wire # "null-element"))
Terminates == <>(phase = "done")
=====================================================================
