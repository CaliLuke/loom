------------------------ MODULE ProjectionOpenness ------------------------
EXTENDS Naturals
CONSTANT Policy
VARIABLES authored, apiClosed, selected, projected, done
vars == <<authored, apiClosed, selected, projected, done>>
Policies == {"unset", "open", "closed"}
Init == /\ authored \in Policies
        /\ apiClosed \in BOOLEAN
        /\ selected \in {{"id"}, {"id", "name"}}
        /\ projected = "unset"
        /\ done = FALSE
Project == /\ ~done
           /\ projected' = CASE Policy = "discard" -> "unset"
                              [] Policy = "collapse" /\ authored = "unset" -> "open"
                              [] OTHER -> authored
           /\ done' = TRUE
           /\ UNCHANGED <<authored, apiClosed, selected>>
Spec == Init /\ [][Project]_vars
Closed(policy) == policy = "closed" \/ (policy = "unset" /\ apiClosed)
AuthoredPolicy == done => projected = authored
RenderedPolicy == done => Closed(projected) = Closed(authored)
=============================================================================
