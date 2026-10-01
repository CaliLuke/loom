----------------------- MODULE OpaqueEnumWitnesses -----------------------
EXTENDS Naturals
CONSTANT Mode
VARIABLES admitted, known, value
vars == <<admitted, known, value>>
Domain == {"a", "b"}
Init ==
    /\ admitted \in [1..2 -> ((SUBSET Domain) \ {{}})]
    /\ known \in [1..2 -> (SUBSET Domain)]
    /\ \A clause \in 1..2: known[clause] \subseteq admitted[clause]
    /\ value \in Domain
\* Opaque alternatives contribute no semantic witness. A clause remains even
\* when every alternative is opaque; otherwise its constraint would disappear.
CheckedEmission ==
    \A clause \in 1..2:
        IF Mode = "drop-empty" /\ known[clause] = {} THEN TRUE
        ELSE value \in known[clause]
SoundMembership == CheckedEmission => \A clause \in 1..2: value \in admitted[clause]
KnownWitnessesRemainUsable ==
    (\A clause \in 1..2: value \in known[clause]) => CheckedEmission
Spec == Init /\ [][UNCHANGED vars]_vars
=============================================================================
