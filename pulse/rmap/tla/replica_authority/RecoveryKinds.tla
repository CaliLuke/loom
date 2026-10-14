--------------------------- MODULE RecoveryKinds ---------------------------
EXTENDS Naturals, Sequences, FiniteSets, TLC
CONSTANT Policy
VARIABLES history, localRev, lastReset, lastDelete, badKind
vars == <<history,localRev,lastReset,lastDelete,badKind>>
Init == /\ history = <<>> /\ localRev = 0 /\ lastReset = 0
        /\ lastDelete = 0 /\ badKind = FALSE
Write == /\ Len(history) < 3
         /\ \E kind \in 1..3:
              /\ history' = Append(history,kind)
              /\ lastReset' = IF kind = 3 THEN Len(history') ELSE lastReset
              /\ lastDelete' = IF kind = 2 THEN Len(history') ELSE lastDelete
         /\ UNCHANGED <<localRev,badKind>>
Observe == /\ localRev < Len(history)
           /\ LET required == IF 3 \in {history[i]: i \in (localRev+1)..Len(history)}
                               THEN 3
                               ELSE IF 2 \in {history[i]: i \in (localRev+1)..Len(history)}
                                    THEN 2 ELSE 1
                  emitted == IF Policy = "Latest" THEN history[Len(history)]
                             ELSE IF lastReset > localRev THEN 3
                                  ELSE IF lastDelete > localRev THEN 2 ELSE 1
              IN badKind' = (badKind \/ emitted < required)
           /\ localRev' = Len(history)
           /\ UNCHANGED <<history,lastReset,lastDelete>>
Next == Write \/ Observe
Spec == Init /\ [][Next]_vars
RetainsUnseenKind == ~badKind
=============================================================================
