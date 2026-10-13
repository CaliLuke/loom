----------------------- MODULE BufferLineage -----------------------
EXTENDS Naturals, FiniteSets, Sequences
CONSTANT Shared
Contexts == 0..2
Children == {1, 2}
VARIABLES buffers, disabled, logged, flushed, counts
vars == <<buffers, disabled, logged, flushed, counts>>
Owner(c) == IF Shared THEN 0 ELSE c
Init == /\ buffers = [c \in Contexts |-> IF Shared /\ c # 0 THEN <<>> ELSE <<0>>]
        /\ disabled = [c \in Contexts |-> FALSE]
        /\ logged = {0} /\ flushed = {}
        /\ counts = [e \in Contexts |-> 0]
Log(c) == /\ c \in Children \ logged
          /\ logged' = logged \cup {c}
          /\ buffers' = IF disabled[Owner(c)] THEN buffers ELSE [buffers EXCEPT ![Owner(c)] = Append(@, c)]
          /\ counts' = IF disabled[Owner(c)] THEN [counts EXCEPT ![c] = @ + 1] ELSE counts
          /\ UNCHANGED <<disabled, flushed>>
Occurrences(seq, e) == Cardinality({i \in DOMAIN seq: seq[i] = e})
Flush(c) == /\ c \notin flushed
            /\ flushed' = flushed \cup {c}
            /\ counts' = [e \in Contexts |-> counts[e] + Occurrences(buffers[Owner(c)], e)]
            /\ buffers' = [buffers EXCEPT ![Owner(c)] = <<>>]
            /\ disabled' = [disabled EXCEPT ![Owner(c)] = TRUE]
            /\ UNCHANGED logged
Next == (\E c \in Children: Log(c)) \/ (\E c \in Contexts: Flush(c))
Spec == Init /\ [][Next]_vars
AtMostOnce == \A e \in Contexts: counts[e] <= 1
CompleteAfterFlush == flushed # {} => \A e \in logged: counts[e] = 1
=============================================================================
