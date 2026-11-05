------------------------- MODULE ForwardedValue -------------------------
EXTENDS Naturals, Sequences, FiniteSets, TLC
CONSTANTS LegacyFirstLine, MaxEntries
VARIABLES entries, firstLineEnd, trustedPeer, selected
vars == <<entries, firstLineEnd, trustedPeer, selected>>
Values == {"client", "proxy", "empty"}
Chains == UNION {[1..n -> Values] : n \in 0..MaxEntries}
Results == {"pending", "original", "client", "proxy"}

\* firstLineEnd abstracts every possible grouping of entries into header lines:
\* Header.Get observes only the first line; Header.Values preserves them all.
Init ==
    /\ entries \in Chains
    /\ firstLineEnd \in IF Len(entries) = 0 THEN {0} ELSE 1..Len(entries)
    /\ trustedPeer \in BOOLEAN
    /\ selected = "pending"

Project(value) == IF value = "empty" THEN "original" ELSE value
Expected ==
    IF ~trustedPeer \/ Len(entries) = 0 THEN "original"
    ELSE Project(entries[Len(entries)])

Read ==
    /\ selected = "pending"
    /\ selected' = IF ~trustedPeer \/ Len(entries) = 0 THEN "original"
                    ELSE Project(entries[IF LegacyFirstLine THEN firstLineEnd ELSE Len(entries)])
    /\ UNCHANGED <<entries, firstLineEnd, trustedPeer>>

Next == Read
TypeOK ==
    /\ entries \in Chains
    /\ firstLineEnd \in 0..Len(entries)
    /\ trustedPeer \in BOOLEAN
    /\ selected \in Results
LastValueSelection == selected # "pending" => selected = Expected
=============================================================================
