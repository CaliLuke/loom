--------------------------- MODULE CLISources ---------------------------
EXTENDS Naturals, Sequences, FiniteSets, TLC
CONSTANT Mode
VARIABLES active, cursor, selected
vars == <<active, cursor, selected>>
Candidates == <<[source |-> "login", local |-> "login", direct |-> TRUE],
                [source |-> "login", local |-> "login2", direct |-> TRUE],
                [source |-> "passcode", local |-> "passcode", direct |-> TRUE],
                [source |-> "body", local |-> "body", direct |-> FALSE],
                [source |-> "login", local |-> "login", direct |-> TRUE],
                [source |-> "passcode", local |-> "passcode", direct |-> TRUE]>>
Init == /\ active \in SUBSET (1..Len(Candidates))
        /\ cursor = 1 /\ selected = <<>>
Seen(arg) == \E j \in 1..Len(selected):
  IF Mode = "printed" THEN selected[j].local = arg.local
  ELSE arg.direct /\ selected[j].direct /\ selected[j].source = arg.source
Step == /\ cursor <= Len(Candidates)
        /\ LET arg == Candidates[cursor]
                keep == cursor \in active /\ (Mode = "append" \/ ~Seen(arg))
            IN selected' = IF keep THEN Append(selected, arg) ELSE selected
        /\ cursor' = cursor + 1
        /\ UNCHANGED active
Next == Step
Spec == Init /\ [][Next]_vars
UniqueDirectSource == \A i,j \in 1..Len(selected):
  (selected[i].direct /\ selected[j].direct /\ selected[i].source = selected[j].source) => i = j
WireCoverage == cursor > Len(Candidates) =>
  \A i \in active: \E j \in 1..Len(selected): selected[j].source = Candidates[i].source
PreserveCarrier == cursor > Len(Candidates) =>
  (4 \in active => \E j \in 1..Len(selected): selected[j] = Candidates[4])
RetainExistingRecord == \A j \in 1..Len(selected):
  \E i \in active: selected[j] = Candidates[i]
=============================================================================
