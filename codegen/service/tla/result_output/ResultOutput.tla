------------------------- MODULE ResultOutput -------------------------
EXTENDS TLC
CONSTANTS Mode, Views
Paths == {"unary", "raw", "stream", "interceptor"}
Sources == {"valid", "nil-object", "nil-collection", "empty-collection",
            "nil-element", "invalid-included", "invalid-excluded"}
VARIABLES path, source, view, serviceError, phase, outcome
vars == <<path, source, view, serviceError, phase, outcome>>
Invalid == source \in {"nil-object", "nil-element", "invalid-included"}
           \/ (source = "invalid-excluded" /\ view = "default")
Contract == view # "unknown" /\ ~Invalid
Init == /\ path \in Paths /\ source \in Sources /\ view \in Views
        /\ serviceError \in BOOLEAN /\ phase = "input" /\ outcome = "pending"
Prepare == /\ phase = "input"
           /\ IF serviceError
              THEN /\ phase' = "done" /\ outcome' = "service-error"
              ELSE IF source = "nil-object"
              THEN /\ phase' = "done" /\ outcome' = "fault"
              ELSE IF view = "unknown"
              THEN /\ phase' = "done"
                   /\ outcome' = IF Mode = "legacy" THEN "caller-error" ELSE "fault"
              ELSE IF source = "nil-element" /\ Mode # "checked"
              THEN /\ phase' = "done" /\ outcome' = "panic"
              ELSE /\ phase' = "projected" /\ UNCHANGED outcome
           /\ UNCHANGED <<path, source, view, serviceError>>
Validate == /\ phase = "projected"
            /\ outcome' = IF Mode = "legacy" \/ ~Invalid THEN "emit" ELSE "fault"
            /\ phase' = "done"
            /\ UNCHANGED <<path, source, view, serviceError>>
Next == Prepare \/ Validate \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
NoCallerBlame == ~serviceError => outcome # "caller-error"
NoPanic == outcome # "panic"
NoBadEmission == outcome = "emit" => Contract
ExactContract == phase = "done" =>
    outcome = (IF serviceError THEN "service-error" ELSE IF Contract THEN "emit" ELSE "fault")
Terminates == <>(phase = "done")
=============================================================================
