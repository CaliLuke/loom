---------------------- MODULE AcquisitionJournal ----------------------
EXTENDS Integers, Sequences, FiniteSets
CONSTANTS Mode, Keys, PriorSize
VARIABLES index, prior, history, phase, writes, work
vars == <<index, prior, history, phase, writes, work>>

RECURSIVE Restore(_, _)
Restore(table, changes) ==
    IF Len(changes) = 0 THEN table
    ELSE LET at == IF Mode = "Forward" THEN 1 ELSE Len(changes)
             change == changes[at]
             rest == IF Mode = "Forward" THEN Tail(changes)
                     ELSE SubSeq(changes, 1, Len(changes) - 1)
         IN Restore([table EXCEPT ![change.key] = change.value], rest)

Init == /\ index = [k \in Keys |-> IF k = "kept" THEN 1 ELSE 0]
        /\ prior = index
        /\ history = <<>>
        /\ phase = "idle"
        /\ writes = 0
        /\ work = 0
Begin == /\ phase = "idle"
         /\ phase' = "active"
         /\ work' = IF Mode = "Dense" THEN PriorSize ELSE 0
         /\ UNCHANGED <<index, prior, history, writes>>
Write(k, value) == /\ phase = "active"
                   /\ writes < 3
                   /\ history' = Append(history, [key |-> k, value |-> index[k]])
                   /\ index' = [index EXCEPT ![k] = value]
                   /\ writes' = writes + 1
                   /\ work' = work + 1
                   /\ UNCHANGED <<prior, phase>>
Abort == /\ phase = "active"
         /\ index' = Restore(index, history)
         /\ phase' = "aborted"
         /\ work' = work + Len(history)
         /\ history' = <<>>
         /\ UNCHANGED <<prior, writes>>
Commit == /\ phase = "active"
          /\ phase' = "committed"
          /\ history' = <<>>
          /\ UNCHANGED <<index, prior, writes, work>>
Next == Begin \/ Abort \/ Commit \/ \E k \in Keys, value \in 0..2 : Write(k, value)
Spec == Init /\ [][Next]_vars
Restored == phase = "aborted" => index = prior
BoundedWork == work <= 2 * writes
Released == phase \in {"aborted", "committed"} => history = <<>>
=============================================================================
