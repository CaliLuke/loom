------------------------ MODULE Observation ------------------------
EXTENDS FiniteSets
CONSTANT Strategy
VARIABLES failed, received, reported, asserted
vars == <<failed, received, reported, asserted>>
Streams == {"a", "b"}

Init == /\ failed = FALSE /\ received = {}
        /\ reported = FALSE /\ asserted = FALSE
Fail == /\ ~failed /\ failed' = TRUE
        /\ UNCHANGED <<received, reported, asserted>>
Receive == /\ failed /\ received # Streams
           /\ \E stream \in Streams \ received : received' = received \cup {stream}
           /\ UNCHANGED <<failed, reported, asserted>>
Report == /\ failed /\ ~reported /\ reported' = TRUE
          /\ UNCHANGED <<failed, received, asserted>>
Assert == /\ received = Streams /\ ~asserted
          /\ (Strategy = "Current" \/ reported)
          /\ asserted' = TRUE
          /\ UNCHANGED <<failed, received, reported>>
Next == Fail \/ Receive \/ Report \/ Assert
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
ObservationIsSynchronized == asserted => reported
Completes == <> asserted
====================================================================
