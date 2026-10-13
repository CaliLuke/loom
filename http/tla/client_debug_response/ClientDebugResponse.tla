------------------------ MODULE ClientDebugResponse ------------------------
EXTENDS Naturals, Sequences, TLC
CONSTANT Mode
VARIABLES started, latest, reads, sourceReads, got, original, terminal, prints,
          owners, steps
vars == <<started, latest, reads, sourceReads, got, original, terminal, prints,
          owners, steps>>
Init == /\ started = {} /\ latest = 0 /\ reads = 0 /\ sourceReads = 0
        /\ got = <<>> /\ original = <<>> /\ terminal = {}
        /\ prints = [i \in 1..2 |-> 0] /\ owners = TRUE /\ steps = 0
Start(i) == /\ i \notin started
 /\ started' = started \cup {i} /\ latest' = i
 /\ sourceReads' = sourceReads + (IF Mode = "eager" THEN 1 ELSE 0)
 /\ UNCHANGED <<reads, got, original, terminal, prints, owners, steps>>
Read(i, err) == /\ i \in started /\ steps < 3
 /\ reads' = reads+1 /\ sourceReads' = sourceReads+1
 /\ original' = Append(original, <<i, err>>)
 /\ got' = Append(got, <<i, IF Mode = "replace" /\ err = "boom" THEN "ok" ELSE err>>)
 /\ steps' = steps+1
 /\ UNCHANGED <<started, latest, terminal, prints, owners>>
Finish(i) == /\ i \in started /\ steps < 3
 /\ terminal' = terminal \cup {i}
 /\ prints' = [prints EXCEPT ![i] = @ + (IF i \notin terminal \/ Mode = "duplicate" THEN 1 ELSE 0)]
 /\ owners' = (owners /\ (Mode # "latest" \/ i = latest \/ i \in terminal))
 /\ steps' = steps+1
 /\ UNCHANGED <<started, latest, reads, sourceReads, got, original>>
Next == (\E i \in 1..2: Start(i) \/ Finish(i) \/
           (\E err \in {"ok", "boom", "eof"}: Read(i, err)))
        \/ UNCHANGED vars
Spec == Init /\ [][Next]_vars
Passive == reads = sourceReads
ReadIdentity == got = original
ExactlyOnce == \A i \in 1..2: prints[i] = (IF i \in terminal THEN 1 ELSE 0)
ExchangeOwnership == owners
=============================================================================
