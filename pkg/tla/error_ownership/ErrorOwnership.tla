--------------------------- MODULE ErrorOwnership ---------------------------
EXTENDS Sequences, TLC
CONSTANTS CopyMerge, CopyHistory, CopyMetadata
VARIABLES first, merged, history, returned, phase
vars == <<first, merged, history, returned, phase>>

Entry(n) == [message |-> <<n>>, field |-> n, remedy |-> n]
Original == Entry(1)
Contributions == <<Original, Entry(2)>>
Summary == [Original EXCEPT !.message = <<1, 2>>]

Init ==
  /\ first = Original
  /\ merged = Original
  /\ history = <<>>
  /\ returned = <<>>
  /\ phase = "merge"

Merge ==
  /\ phase = "merge"
  /\ first' = IF CopyMerge THEN first ELSE Summary
  /\ merged' = Summary
  /\ history' = Contributions
  /\ phase' = "read"
  /\ UNCHANGED returned

Read ==
  /\ phase = "read"
  /\ returned' = history
  /\ phase' = "mutate-read"
  /\ UNCHANGED <<first, merged, history>>

MutateRead(member) ==
  /\ phase = "mutate-read"
  /\ returned' = [returned EXCEPT ![1][member] =
       IF member = "message" THEN <<3>> ELSE 3]
  /\ history' = IF ~CopyHistory \/ (member # "message" /\ ~CopyMetadata)
                  THEN returned' ELSE history
  /\ phase' = "mutate-result"
  /\ UNCHANGED <<first, merged>>

MutateResult(member) ==
  /\ phase = "mutate-result"
  /\ merged' = [merged EXCEPT ![member] = 3]
  /\ first' = IF CopyMetadata THEN first ELSE [first EXCEPT ![member] = 3]
  /\ phase' = "done"
  /\ UNCHANGED <<history, returned>>

Next == Merge \/ Read
        \/ (\E member \in {"message", "field", "remedy"}: MutateRead(member))
        \/ (\E member \in {"field", "remedy"}: MutateResult(member))
Spec == Init /\ [][Next]_vars
InputsRetained == first = Original
HistoryRetained == phase = "merge" \/ history = Contributions
=============================================================================
