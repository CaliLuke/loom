---------------------------- MODULE MediaBodyCopy ----------------------------
EXTENDS Naturals
CONSTANT Refresh
VARIABLES source, body, variant, phase
vars == <<source, body, variant, phase>>
Init == /\ source = 0 /\ body = 0 /\ variant = 0 /\ phase = "copy"
Copy == /\ phase = "copy" /\ body' = 1
        /\ variant' = IF Refresh THEN 1 ELSE variant
        /\ phase' = "done" /\ UNCHANGED source
Next == Copy
Spec == Init /\ [][Next]_vars
SourceUnchanged == source = 0
VariantOwnsEmittedBranch == phase = "done" => variant = body
=============================================================================
