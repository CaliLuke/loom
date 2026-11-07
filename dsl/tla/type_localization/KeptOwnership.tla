-------------------------- MODULE KeptOwnership --------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Policy
VARIABLES sourceRole, copiedRole, targets, completed, requirements
vars == <<sourceRole, copiedRole, targets, completed, requirements>>
Occurrences == {"left", "right"}
Roles == {"structural", "localized"}
Owner == "declaration"
Local(o) == IF o = "left" THEN "left-copy" ELSE "right-copy"
Init == /\ sourceRole \in [Occurrences -> Roles]
        /\ copiedRole = [o \in Occurrences |-> IF Policy = "snapshot" THEN "unknown" ELSE sourceRole[o]]
        /\ targets = [o \in Occurrences |-> "pending"]
        /\ completed = {}
        /\ requirements = [o \in Occurrences |-> IF o = "left" THEN {"v"} ELSE {}]
Complete(o) == /\ o \notin completed
               /\ targets' = [targets EXCEPT ![o] =
                    IF Policy = "bind-all" \/ copiedRole[o] = "structural"
                    THEN Owner ELSE Local(o)]
               /\ completed' = completed \cup {o}
               /\ UNCHANGED <<sourceRole, copiedRole, requirements>>
Next == (\E o \in Occurrences: Complete(o)) \/
        (completed = Occurrences /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
StructuralClosure == \A o \in completed: sourceRole[o] = "structural" => targets[o] = Owner
LocalizedIsolation == \A o \in completed: sourceRole[o] = "localized" => targets[o] = Local(o)
OccurrenceRequirements == requirements = [o \in Occurrences |-> IF o = "left" THEN {"v"} ELSE {}]
=============================================================================
