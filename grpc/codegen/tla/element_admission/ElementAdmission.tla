----------------------- MODULE ElementAdmission -----------------------
EXTENDS Naturals, FiniteSets

CONSTANT Mode
VARIABLES kind, position, shape, direction, boundary, phase, outcome, sent

Kinds == {"nil-object", "empty-object", "populated-object", "nil-collection",
          "empty-collection", "populated-collection", "any-null", "scalar-zero"}
Positions == {"array", "map", "optional-field"}
Shapes == {"inline", "recursive-helper", "empty-declaration"}
Directions == {"service-to-proto", "proto-to-service"}
Boundaries == {"unary", "stream", "error-detail"}
Invalid == kind = "nil-object" /\ position # "optional-field"
NilStorage == kind \in {"nil-object", "nil-collection", "any-null"}
vars == <<kind, position, shape, direction, boundary, phase, outcome, sent>>

Init == /\ kind \in Kinds
        /\ position \in Positions
        /\ shape \in Shapes
        /\ direction \in Directions
        /\ boundary \in Boundaries
        /\ phase = "conversion"
        /\ outcome = "pending"
        /\ sent = FALSE

Convert == /\ phase = "conversion"
           /\ phase' = "boundary"
           /\ outcome' =
                IF Mode = "blanket-nil" /\ NilStorage THEN "error"
                ELSE IF Mode = "owner" /\ Invalid THEN "error"
                ELSE IF Mode = "current" /\ Invalid /\ shape = "inline"
                     THEN "panic"
                ELSE IF Invalid /\ shape # "empty-declaration" THEN "nil"
                ELSE "value"
           /\ UNCHANGED <<kind, position, shape, direction, boundary, sent>>

Publish == /\ phase = "boundary"
           /\ phase' = "done"
           /\ sent' = (outcome \notin {"error", "panic"})
           /\ outcome' = IF outcome = "nil" /\ direction = "service-to-proto"
                         THEN "empty-message" ELSE outcome
           /\ UNCHANGED <<kind, position, shape, direction, boundary>>

Next == Convert \/ Publish
Spec == Init /\ [][Next]_vars

NoPanic == outcome # "panic"
NoInvalidEmission == phase = "done" /\ Invalid => ~sent
ExactAdmission == phase = "done" => (sent <=> ~Invalid)
=======================================================================
