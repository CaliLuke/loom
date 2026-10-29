---------------------- MODULE MediaContracts ----------------------
EXTENDS Naturals, FiniteSets, Sequences
CONSTANT Collapse
Contracts == <<"*/*", "text/*", "text/plain", "application/json">>
Headers == {"text/plain", "text/html", "application/json", "image/png"}
Rank(i, h) == IF Contracts[i] = h THEN 3 ELSE
  IF Contracts[i] = "text/*" /\ h \in {"text/plain", "text/html"} THEN 2 ELSE
  IF Contracts[i] = "*/*" THEN 1 ELSE 0
Best(h) == CHOOSE i \in 1..Len(Contracts): \A j \in 1..Len(Contracts): Rank(i,h) >= Rank(j,h)
VARIABLE header, selected, phase
vars == <<header, selected, phase>>
Init == header = "" /\ selected = 0 /\ phase = "idle"
Receive == phase = "idle" /\ \E h \in Headers:
  header' = h /\ selected' = (IF Collapse THEN 1 ELSE Best(h)) /\ phase' = "decoded"
Reply == phase = "decoded" /\ phase' = "published" /\ UNCHANGED <<header, selected>>
Next == Receive \/ Reply
Specificity == phase # "idle" => selected = Best(header)
Concrete == phase = "published" => header \in Headers
Spec == Init /\ [][Next]_vars
==================================================================
