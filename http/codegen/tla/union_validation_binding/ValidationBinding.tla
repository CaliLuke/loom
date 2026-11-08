-------------------------- MODULE ValidationBinding --------------------------
EXTENDS Naturals
CONSTANT Policy
VARIABLES first, inherited, named, step, allocated, published, graph, operand
vars == <<first, inherited, named, step, allocated, published, graph, operand>>
Init == /\ first \in BOOLEAN /\ inherited \in BOOLEAN /\ named \in BOOLEAN /\ step = 0
        /\ allocated = first
        /\ published = [package \in BOOLEAN |-> "unset"]
        /\ graph = [package \in BOOLEAN |-> "unset"]
        /\ operand = [package \in BOOLEAN |-> "unset"]
Next == /\ step < 2 /\ step' = step + 1
        /\ LET package == IF step = 0 THEN first ELSE ~first
               binding == CASE Policy = "default" -> FALSE
                            [] Policy = "shared" -> allocated
                            [] Policy = "root" -> ~package
                            [] OTHER -> package
           IN /\ published' = [published EXCEPT ![package] = binding]
              /\ graph' = [graph EXCEPT ![package] = IF Policy = "stale-json" THEN allocated ELSE binding]
              /\ operand' = [operand EXCEPT ![package] = IF Policy = "unadapted" \/ named THEN binding ELSE FALSE]
        /\ UNCHANGED <<first, inherited, named, allocated>>
OwnDeclaration == step = 2 => \A package \in BOOLEAN : published[package] = package
JSONAgreement == \A package \in BOOLEAN : graph[package] = published[package]
RootStorage == step = 2 => \A package \in BOOLEAN : operand[package] = IF named THEN package ELSE FALSE
Spec == Init /\ [][Next]_vars
=============================================================================
