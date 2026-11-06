----------------------- MODULE RequiredOccurrence -----------------------
EXTENDS TLC
CONSTANT Policy
VARIABLES declaration, local, copied, done, completed
vars == <<declaration, local, copied, done, completed>>
Owners == {"left", "right"}
Intrinsic == {"id"}
Authored(owner) == IF owner = "left" THEN {"left"} ELSE {"right"}
Init == /\ declaration = Intrinsic
        /\ local = [o \in Owners |-> {}]
        /\ copied = [o \in Owners |-> {}]
        /\ done = {} /\ completed = {}
Refine(owner) == /\ owner \notin done
                 /\ local' = [local EXCEPT ![owner] = Authored(owner)]
                 /\ declaration' = IF Policy = "write-through"
                                    THEN declaration \cup Authored(owner)
                                    ELSE declaration
                 /\ done' = done \cup {owner}
                 /\ UNCHANGED <<copied, completed>>
Complete(owner) == /\ owner \in done \ completed
                   /\ copied' = [copied EXCEPT ![owner] =
                         IF Policy = "occurrence" THEN declaration
                         ELSE declaration \cup local[owner]]
                   /\ completed' = completed \cup {owner}
                   /\ UNCHANGED <<declaration, local, done>>
Next == (\E o \in Owners: Refine(o) \/ Complete(o))
        \/ (completed = Owners /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
SharedUnchanged == declaration = Intrinsic
DeclarationStorageUnchanged == \A o \in completed: copied[o] = Intrinsic
OccurrenceConjunction == \A o \in completed:
                          copied[o] \cup local[o] = Intrinsic \cup Authored(o)
=============================================================================
