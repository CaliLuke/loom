-------------------------- MODULE UntaggedJSON --------------------------
EXTENDS Naturals, FiniteSets, Sequences
CONSTANT Mode
Branches == {1, 2}
VARIABLES schema, decoded, fatal, selected, destination, emitted, done
vars == <<schema, decoded, fatal, selected, destination, emitted, done>>

Init == /\ schema \in SUBSET Branches
        /\ decoded \in SUBSET Branches
        /\ fatal \in BOOLEAN
        /\ selected \in Branches
        /\ destination = 0
        /\ emitted = FALSE
        /\ done = FALSE

Unique == /\ Cardinality(schema) = 1
          /\ Cardinality(decoded) = 1
          /\ schema = decoded
Accepted == /\ Unique /\ ~fatal
Step == /\ ~done
        /\ done' = TRUE
        /\ UNCHANGED <<schema, decoded, fatal, selected>>
        /\ IF Mode = "Legacy"
              THEN /\ emitted' = (selected \in decoded)
                   /\ destination' = IF Cardinality(decoded) = 1
                                      THEN CHOOSE b \in decoded : TRUE ELSE 0
              ELSE /\ emitted' = (Accepted /\ selected \in decoded)
                   /\ destination' = IF Accepted THEN CHOOSE b \in decoded : TRUE ELSE 0

Next == Step
Spec == Init /\ [][Next]_vars
EmissionRetainsIdentity == emitted => Accepted /\ selected \in schema
FailureIsAtomic == done /\ ~Accepted => destination = 0
SuccessAgrees == destination # 0 => Accepted /\ destination \in schema
=============================================================================
