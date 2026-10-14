-------------------------- MODULE ReleaseFaultOwner --------------------------
EXTENDS Naturals, FiniteSets, TLC
CONSTANT Owned
Actors == {"direct", "async", "other"}
VARIABLES phase, fired, admitted, mainTaken
vars == <<phase, fired, admitted, mainTaken>>
Init == /\ phase = [a \in Actors |-> "idle"]
        /\ fired = FALSE
        /\ admitted = {}
        /\ mainTaken = FALSE
Enter(a) == /\ phase[a] = "idle"
            /\ IF a = "other" THEN TRUE
               ELSE /\ ~mainTaken
                    /\ (a = "direct" \/ ~Owned)
            /\ phase' = [phase EXCEPT ![a] = "release"]
            /\ mainTaken' = IF a = "other" THEN mainTaken ELSE TRUE
            /\ UNCHANGED <<fired, admitted>>
Admit(a) == /\ phase[a] = "release"
            /\ ~fired
            /\ (~Owned \/ a = "direct")
            /\ admitted' = admitted \cup {a}
            /\ phase' = [phase EXCEPT ![a] = "gated"]
            /\ fired' = IF Owned THEN TRUE ELSE fired
            /\ UNCHANGED mainTaken
Fault(a) == /\ phase[a] = "gated"
            /\ phase' = [phase EXCEPT ![a] = "error"]
            /\ fired' = TRUE
            /\ UNCHANGED <<admitted, mainTaken>>
Pass(a) == /\ phase[a] = "release"
           /\ (fired \/ (Owned /\ a # "direct"))
           /\ phase' = [phase EXCEPT ![a] = "success"]
           /\ UNCHANGED <<fired, admitted, mainTaken>>
RetryDirect == /\ phase["direct"] = "idle"
               /\ phase["async"] \in {"error", "success"}
               /\ phase' = [phase EXCEPT !["direct"] = "success"]
               /\ UNCHANGED <<fired, admitted, mainTaken>>
Next == (\E a \in Actors : Enter(a) \/ Admit(a) \/ Fault(a) \/ Pass(a))
        \/ RetryDirect
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
FaultBelongsToDirect == admitted \subseteq {"direct"}
GateHasOneOwner == Cardinality(admitted) <= 1
DirectFaultEventually == <> (phase["direct"] = "error")
=============================================================================
