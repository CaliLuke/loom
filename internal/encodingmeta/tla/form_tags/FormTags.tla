------------------------------ MODULE FormTags ------------------------------
EXTENDS Naturals
CONSTANT Policy
VARIABLES tag, peer, names, accepted, phase
vars == <<tag, peer, names, accepted, phase>>
Fields == {"value", "peer"}
Wire == IF tag = "default" THEN "value" ELSE IF tag = "rename" THEN "wire" ELSE ""
Expected == [f \in Fields |-> IF f = "value" THEN Wire ELSE peer]
Disjoint(n) == n["value"] = "" \/ n["peer"] = "" \/ n["value"] # n["peer"]
Init == /\ tag \in {"default", "rename", "empty", "skip", "json-only"}
        /\ peer \in {"wire", "value", "other"}
        /\ names = [f \in Fields |-> f] /\ accepted = FALSE /\ phase = "capture"
Capture == /\ phase = "capture"
           /\ names' = IF Policy = "element" THEN [f \in Fields |-> IF f = "value" THEN "value" ELSE peer] ELSE Expected
           /\ accepted' = Disjoint(IF Policy = "namespace-element" THEN [f \in Fields |-> IF f = "value" THEN "value" ELSE peer] ELSE names')
           /\ phase' = "done" /\ UNCHANGED <<tag, peer>>
Next == Capture \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
WireAgreement == phase = "done" => names = Expected
NamespaceAgreement == phase = "done" => accepted = Disjoint(Expected)
=============================================================================
