---------------- MODULE ScalarBindingPresence ----------------
EXTENDS Naturals
CONSTANT ExplicitPresence
VARIABLES supplied, empty, required, defaulted, codecAcceptsEmpty
vars == <<supplied, empty, required, defaulted, codecAcceptsEmpty>>
Init == /\ supplied \in BOOLEAN
        /\ empty \in BOOLEAN
        /\ required \in BOOLEAN
        /\ defaulted \in BOOLEAN
        /\ codecAcceptsEmpty \in BOOLEAN
Observed == supplied /\ (ExplicitPresence \/ ~empty)
Missing == required /\ ~Observed
Convert == Observed
Default == ~required /\ ~Observed /\ defaulted
RejectedByCodec == Convert /\ empty /\ ~codecAcceptsEmpty
Accepted == ~Missing /\ ~RejectedByCodec
Presence == Observed = supplied
DefaultsOnlyAbsent == Default => ~supplied
RequiredAdmission == Missing = (required /\ ~supplied)
ConversionAdmission == Convert = supplied
Next == UNCHANGED vars
Spec == Init /\ [][Next]_vars
===============================================================
