-------------------------- MODULE FieldFacts --------------------------
EXTENDS Naturals, FiniteSets
CONSTANTS Fields, Descendants, Mode
VARIABLES source, snapshot, original, queried, answers, work, mutated
vars == <<source, snapshot, original, queried, answers, work, mutated>>
Init == /\ source = [f \in Fields |-> FALSE]
        /\ snapshot = source
        /\ original = source
        /\ queried = {}
        /\ answers = [f \in Fields |-> FALSE]
        /\ work = Cardinality(Fields)
        /\ mutated = FALSE
Mutate == /\ ~mutated
          /\ source' = [f \in Fields |-> TRUE]
          /\ mutated' = TRUE
          /\ UNCHANGED <<snapshot, original, queried, answers, work>>
Query(f) == /\ f \notin queried
            /\ queried' = queried \cup {f}
            /\ answers' = [answers EXCEPT ![f] = IF Mode = "Borrowed" THEN source[f] ELSE snapshot[f]]
            /\ work' = work + IF Mode = "Copied" THEN Cardinality(Fields) * Descendants ELSE 0
            /\ UNCHANGED <<source, snapshot, original, mutated>>
Next == Mutate \/ \E f \in Fields : Query(f)
Spec == Init /\ [][Next]_vars
SnapshotMeaning == \A f \in queried : answers[f] = original[f]
NoRepeatedGraphWork == work <= Cardinality(Fields)
=============================================================================
