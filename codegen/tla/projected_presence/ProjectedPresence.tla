----------------------- MODULE ProjectedPresence -----------------------
EXTENDS TLC
CONSTANTS Mode, Kinds
VARIABLES kind, required, selected, knownView, supplied, phase, carrier, panic, accepted
vars == <<kind, required, selected, knownView, supplied, phase, carrier, panic, accepted>>
Init == /\ kind \in Kinds
        /\ required \in BOOLEAN
        /\ selected \in BOOLEAN
        /\ knownView \in BOOLEAN
        /\ supplied \in BOOLEAN
        /\ phase = "decoded"
        /\ carrier = FALSE /\ panic = FALSE /\ accepted = FALSE
Convert == /\ phase = "decoded"
           /\ LET guard == ~required \/ Mode = "preserve"
                          \/ (Mode = "objects-only" /\ kind \in {"object", "scalar"})
                  dereference == kind \in {"object", "scalar"}
              IN /\ panic' = (~supplied /\ ~guard /\ dereference)
                 /\ carrier' = (supplied \/ (~guard /\ ~dereference))
           /\ phase' = "converted"
           /\ UNCHANGED <<kind, required, selected, knownView, supplied, accepted>>
Validate == /\ phase = "converted"
            /\ accepted' = (~panic /\ knownView /\ (~selected \/ ~required \/ carrier))
            /\ phase' = "done"
            /\ UNCHANGED <<kind, required, selected, knownView, supplied, carrier, panic>>
Next == Convert \/ Validate \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
NoPanic == ~panic
PresencePreserved == phase # "decoded" => carrier = supplied
SelectedContract == phase = "done" =>
    accepted = (knownView /\ (~selected \/ ~required \/ supplied))
Terminates == <>(phase = "done")
=============================================================================
