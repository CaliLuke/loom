--------------------- MODULE SelectedSourceStorage ---------------------
EXTENDS TLC
CONSTANT Mode
VARIABLE specimen, present, readPointer, bodyPointer, bodyPresent, complete
vars == <<specimen, present, readPointer, bodyPointer, bodyPresent, complete>>

\* Only source fields needing a scalar body constructor are represented.
\* Explicit presence wrappers carry their own absent/null state, not a pointer.
Cases == {
    [name |-> "required named view", pointer |-> TRUE, optional |-> FALSE],
    [name |-> "optional named view", pointer |-> TRUE, optional |-> TRUE],
    [name |-> "optional named service", pointer |-> TRUE, optional |-> TRUE],
    [name |-> "required named service", pointer |-> FALSE, optional |-> FALSE],
    [name |-> "nullable wrapper", pointer |-> FALSE, optional |-> FALSE]
}
Init == /\ specimen \in Cases
        /\ present \in (IF specimen.optional THEN BOOLEAN ELSE {TRUE})
        /\ readPointer = FALSE
        /\ bodyPointer = FALSE
        /\ bodyPresent = FALSE
        /\ complete = FALSE
Convert == /\ ~complete
           /\ readPointer' = IF Mode = "current" THEN FALSE
                              ELSE specimen.pointer /\ (Mode = "dereference" \/ present)
           /\ bodyPointer' = IF Mode = "presence" THEN specimen.pointer ELSE FALSE
           /\ bodyPresent' = IF Mode = "presence" THEN present ELSE TRUE
           /\ complete' = TRUE
           /\ UNCHANGED <<specimen, present>>
Next == Convert
Spec == Init /\ [][Next]_vars
SourceReadMatchesStorage == complete /\ present => readPointer = specimen.pointer
NoAbsentRead == complete /\ ~present => ~readPointer
PreserveAbsence == complete /\ ~present => bodyPointer /\ ~bodyPresent
=============================================================================
