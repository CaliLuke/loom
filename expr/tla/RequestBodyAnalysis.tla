----------------------- MODULE RequestBodyAnalysis -----------------------
EXTENDS Naturals, TLC
CONSTANT PureAnalysis
VARIABLES body, validations, finalized
vars == <<body, validations, finalized>>
Endpoints == {"first", "second"}
Init == /\ body = [e \in Endpoints |-> 0]
        /\ validations = [e \in Endpoints |-> 0]
        /\ finalized = {}
Analyze(e) == /\ e \notin finalized /\ validations[e] < 2
              /\ validations' = [validations EXCEPT ![e] = @ + 1]
              /\ body' = IF PureAnalysis THEN body ELSE [body EXCEPT ![e] = @ + 1]
              /\ UNCHANGED finalized
Finalize(e) == /\ e \notin finalized
               /\ body' = [body EXCEPT ![e] = @ + 1]
               /\ finalized' = finalized \cup {e}
               /\ UNCHANGED validations
Next == \E e \in Endpoints: Analyze(e) \/ Finalize(e)
Spec == Init /\ [][Next]_vars
RenamedOnce == \A e \in finalized: body[e] = 1
=============================================================================
