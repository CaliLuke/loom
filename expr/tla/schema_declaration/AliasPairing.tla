---------------------------- MODULE AliasPairing ----------------------------
EXTENDS Naturals, Sequences, FiniteSets
CONSTANTS Mode, Depth, MaxTargetLength, InputDomain
VARIABLES targetOrigins, cursor, position, role, rejected, structuralChecks
vars == <<targetOrigins, cursor, position, role, rejected, structuralChecks>>

\* Source aliases have distinct immutable origin labels 0..Depth. Targets
\* may repeat an origin (insert a wrapper), skip source alias levels, or contain
\* a foreign/backwards origin that must be rejected. The last target is structural:
\* a controlled flattened copy can retain any outer alias origin while replacing
\* its Type with the underlying structure. An explicit body can instead have an
\* independent structural definition, authorized by the root binding and checked
\* kind/member/branch mapping. Labels are not names or hashes.
Targets == UNION {[1..n -> 0..(Depth + 1)] : n \in 2..MaxTargetLength}
Init ==
    /\ targetOrigins \in {t \in Targets : t[1] = 0}
    /\ InputDomain = "valid" =>
        /\ \A i \in 1..Len(targetOrigins) : targetOrigins[i] <= Depth
        /\ \A i \in 2..Len(targetOrigins) : targetOrigins[i - 1] <= targetOrigins[i]
    /\ structuralChecks \in IF InputDomain = "valid" THEN {TRUE} ELSE BOOLEAN
    /\ cursor = 0
    /\ position = 1
    /\ role = "root"
    /\ rejected = FALSE

Matches(origin) == {i \in cursor..Depth : i = origin}
Nearest(matches) == CHOOSE i \in matches : \A j \in matches : i <= j
Step ==
    /\ ~rejected
    /\ position < Len(targetOrigins)
    /\ LET nextOrigin == targetOrigins[position + 1]
           matches == Matches(nextOrigin)
           structural == position + 1 = Len(targetOrigins)
           legacy == IF cursor < Depth THEN cursor + 1 ELSE cursor
           aligned == IF Mode = "legacy" THEN legacy
                      ELSE IF matches = {} THEN cursor ELSE Nearest(matches)
           child == IF structural /\ Mode # "ancestry-only"
                    THEN Depth ELSE aligned
       IN /\ cursor' = child
          /\ rejected' = IF structural
                          THEN ~structuralChecks \/ (Mode = "terminal-ancestry" /\ matches = {})
                          ELSE Mode # "legacy" /\ matches = {}
          /\ role' = IF structural /\ matches = {} THEN "mapped"
                      ELSE IF aligned = cursor THEN "representation" ELSE "authored"
    /\ position' = position + 1
    /\ UNCHANGED <<targetOrigins, structuralChecks>>
Spec == Init /\ [][Step]_vars

\* Expected source identity comes from captured input ancestry, independently
\* of the selected cursor. A valid prefix cannot silently skip or swap origins.
ValidPrefix ==
    LET namedEnd == IF position = Len(targetOrigins) THEN position - 1 ELSE position
    IN /\ \A i \in 1..namedEnd : targetOrigins[i] <= Depth
       /\ \A i \in 2..namedEnd : targetOrigins[i - 1] <= targetOrigins[i]
       /\ position = Len(targetOrigins) => structuralChecks
SourceCorrespondence ==
    ~rejected => cursor = IF position = Len(targetOrigins) THEN Depth ELSE targetOrigins[position]
ExactRejection == rejected = ~ValidPrefix
EdgeAuthority ==
    (~rejected /\ position > 1) =>
        (role = "representation") = (targetOrigins[position] = targetOrigins[position - 1])
=============================================================================
