---------------------- MODULE CollectionContents ----------------------
EXTENDS Naturals, TLC
CONSTANT Mode
VARIABLES kind, position, present, minimum, size, valuePresent, phase, childPresent, accepted, output
vars == <<kind, position, present, minimum, size, valuePresent, phase, childPresent, accepted, output>>
Init == /\ kind \in {"collection", "scalar", "union"}
        /\ position \in {"root", "named", "nested"}
        /\ present \in BOOLEAN
        /\ minimum \in {0, 1}
        /\ size \in {0, 1}
        /\ valuePresent \in BOOLEAN
        /\ phase = "source"
        /\ childPresent = FALSE
        /\ accepted = FALSE
        /\ output = "absent"
EncodeDecode == /\ phase = "source"
                /\ childPresent' = (present /\ (IF kind = "collection" THEN size > 0 ELSE valuePresent))
                /\ phase' = "decoded"
                /\ UNCHANGED <<kind, position, present, minimum, size, valuePresent, accepted, output>>
RequiredContent == CASE Mode = "drop-all" -> FALSE
                    [] Mode = "current" -> kind # "collection" \/ position = "nested"
                    [] OTHER -> kind # "collection"
Validate == /\ phase = "decoded"
            /\ accepted' = (present /\ (~RequiredContent \/ childPresent)
                            /\ (kind # "collection" \/ size >= minimum))
            /\ phase' = "validated"
            /\ UNCHANGED <<kind, position, present, minimum, size, valuePresent, childPresent, output>>
Convert == /\ phase = "validated"
           /\ output' = IF accepted THEN IF kind = "collection" /\ size = 0
                                         THEN "present-empty" ELSE "present-value"
                        ELSE "absent"
           /\ phase' = "done"
           /\ UNCHANGED <<kind, position, present, minimum, size, valuePresent, childPresent, accepted>>
Next == EncodeDecode \/ Validate \/ Convert
Spec == Init /\ [][Next]_vars
Expected == present /\ (IF kind = "collection" THEN size >= minimum ELSE valuePresent)
Admission == phase \in {"validated", "done"} => accepted = Expected
EmptyPreserved == phase = "done" /\ kind = "collection" /\ present /\ size = 0 /\ minimum = 0
                  => output = "present-empty"
RequiredMessage == phase \in {"validated", "done"} /\ ~present => ~accepted
=============================================================================
