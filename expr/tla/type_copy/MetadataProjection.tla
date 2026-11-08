-------------------------- MODULE MetadataProjection -------------------------
EXTENDS Naturals, FiniteSets
CONSTANT CaptureBeforeRemoval
VARIABLES pending, captured, removed, source, declaration
vars == <<pending, captured, removed, source, declaration>>
Occurrences == {"inherited", "overridden"}
Keys == {"hint", "sourceType", "unrelated"}
Original == [key \in Keys |-> IF key = "hint" THEN "base" ELSE key]
Expected(o, key) == IF o = "overridden" /\ key = "hint" THEN "local" ELSE Original[key]
Init == /\ pending = Occurrences
        /\ captured = [o \in Occurrences |-> [key \in Keys |-> IF o = "overridden" /\ key = "hint" THEN "local" ELSE "absent"]]
        /\ removed = {}
        /\ source = Original
        /\ declaration = [o \in Occurrences |-> "Authored"]
Remove(o) == /\ o \in pending
             /\ pending' = pending \ {o}
             /\ removed' = removed \cup {o}
             /\ captured' = IF CaptureBeforeRemoval
                   THEN [captured EXCEPT ![o] = [key \in Keys |-> IF key = "unrelated" THEN "absent" ELSE Expected(o, key)]]
                   ELSE captured
             /\ UNCHANGED <<source, declaration>>
Next == \E o \in pending : Remove(o)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
ContractsRetained == \A o \in removed : \A key \in Keys \ {"unrelated"} : captured[o][key] = Expected(o, key)
NoUnrelatedInheritance == \A o \in removed : captured[o]["unrelated"] = "absent"
SourceUnchanged == source = Original
DeclarationUnchanged == \A o \in Occurrences : declaration[o] = "Authored"
Terminates == <>(pending = {})
=============================================================================
