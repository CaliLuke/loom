------------------------- MODULE ForwardedClient -------------------------
EXTENDS Naturals, Sequences, FiniteSets, TLC
CONSTANTS LegacyFiltering, MaxHops
VARIABLES chain, alternate, trustedPeer, cursor, selected, alternateSelected
vars == <<chain, alternate, trustedPeer, cursor, selected, alternateSelected>>

Hops == {"trusted", "clientA", "clientB", "invalid"}
Chains == UNION {[1..n -> Hops] : n \in 0..MaxHops}
Results == {"pending", "peer", "clientA", "clientB"}

\* The rightmost non-trusted entry owns the selection boundary. Entries
\* left of it may be changed arbitrarily without changing the result.
Boundary(s) ==
    LET stops == {i \in 1..Len(s) : s[i] # "trusted"}
    IN IF stops = {} THEN 0
       ELSE CHOOSE i \in stops : \A j \in stops : i >= j
SameBoundary(s, other) ==
    /\ Len(s) = Len(other)
    /\ IF Boundary(s) = 0 THEN s = other
       ELSE \A i \in Boundary(s)..Len(s) : s[i] = other[i]
Expected(s) ==
    IF ~trustedPeer \/ Boundary(s) = 0 THEN "peer"
    ELSE IF s[Boundary(s)] = "invalid" THEN "peer"
         ELSE s[Boundary(s)]

Init ==
    /\ chain \in Chains
    /\ alternate \in {s \in Chains : SameBoundary(chain, s)}
    /\ trustedPeer \in BOOLEAN
    /\ cursor = IF trustedPeer THEN Len(chain) ELSE 0
    /\ selected = "pending"
    /\ alternateSelected = "pending"

ReadHop(hop, result) ==
    IF result # "pending" THEN result
    ELSE IF hop = "trusted" \/ (LegacyFiltering /\ hop = "invalid")
         THEN "pending"
         ELSE IF hop = "invalid" THEN "peer" ELSE hop

Read ==
    /\ cursor > 0
    /\ (selected = "pending" \/ alternateSelected = "pending")
    /\ selected' = ReadHop(chain[cursor], selected)
    /\ alternateSelected' = ReadHop(alternate[cursor], alternateSelected)
    /\ cursor' = cursor - 1
    /\ UNCHANGED <<chain, alternate, trustedPeer>>

Fallback ==
    /\ cursor = 0
    /\ (selected = "pending" \/ alternateSelected = "pending")
    /\ selected' = IF selected = "pending" THEN "peer" ELSE selected
    /\ alternateSelected' = IF alternateSelected = "pending" THEN "peer" ELSE alternateSelected
    /\ UNCHANGED <<chain, alternate, trustedPeer, cursor>>

Next == Read \/ Fallback
TypeOK ==
    /\ chain \in Chains /\ alternate \in Chains
    /\ trustedPeer \in BOOLEAN /\ cursor \in 0..Len(chain)
    /\ selected \in Results /\ alternateSelected \in Results
PrefixIndependent ==
    (selected # "pending" /\ alternateSelected # "pending") => selected = alternateSelected
BoundarySelection ==
    /\ selected # "pending" => selected = Expected(chain)
    /\ alternateSelected # "pending" => alternateSelected = Expected(alternate)
=============================================================================
