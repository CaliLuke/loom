--------------------------- MODULE PackageBinding ---------------------------
EXTENDS Naturals
CONSTANT Detached
VARIABLES first, step, shared, published
vars == <<first, step, shared, published>>
Init == /\ first \in BOOLEAN /\ step = 0 /\ shared = first
        /\ published = [package \in BOOLEAN |-> "unset"]
Next == /\ step < 2 /\ step' = step + 1
        /\ LET package == IF step = 0 THEN first ELSE ~first
               binding == IF Detached THEN package ELSE shared
           IN published' = [published EXCEPT ![package] = binding]
        /\ UNCHANGED <<first, shared>>
OwnContext == step = 2 => \A package \in BOOLEAN : published[package] = package
Spec == Init /\ [][Next]_vars
=============================================================================
