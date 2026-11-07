-------------------------- MODULE NullableErrorOwner --------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Policy
VARIABLES stage, presence, declared, selector, candidates, bodyless, metadata,
          ownerIsError, wirePresence, matches, accepted
vars == <<stage, presence, declared, selector, candidates, bodyless, metadata,
          ownerIsError, wirePresence, matches, accepted>>
Names == {"a", "b"}
States == {"absent", "null", "value"}
Init == /\ stage = "new"
        /\ presence \in States
        /\ declared \in Names
        /\ selector \in Names \cup {"none"}
        /\ candidates \in { {declared}, Names }
        /\ bodyless \in BOOLEAN
        /\ metadata \in BOOLEAN
        /\ (metadata => bodyless)
        /\ ownerIsError = FALSE
        /\ wirePresence = "none"
        /\ matches = {}
        /\ accepted = FALSE
Create == /\ stage = "new"
          /\ ownerIsError' = (Policy # "payload-owner")
          /\ stage' = "owned"
          /\ UNCHANGED <<presence, declared, selector, candidates, bodyless, metadata,
                          wirePresence, matches, accepted>>
SelectorAgrees == presence # "value" \/ selector = "none" \/ selector = declared
Encode == /\ stage = "owned"
          /\ IF (SelectorAgrees \/ Policy = "discard-selector") /\
                (presence # "absent" \/ bodyless) /\
                (~metadata \/ presence = "value" \/ Policy = "bodyless-all")
             THEN /\ stage' = "encoded"
                  /\ wirePresence' = IF Policy = "collapse-null" /\ presence = "null"
                                     THEN "absent" ELSE presence
             ELSE /\ stage' = "rejected"
                  /\ UNCHANGED wirePresence
          /\ UNCHANGED <<presence, declared, selector, candidates, bodyless, metadata,
                          ownerIsError, matches, accepted>>
Eligible == IF wirePresence = "absent" /\ selector # "none" THEN {}
            ELSE IF wirePresence = "value" /\ selector # "none"
            THEN {n \in candidates : n = selector}
            ELSE candidates
Decode == /\ stage = "encoded"
          /\ matches' = Eligible
          /\ accepted' = IF Policy = "first-match" THEN Cardinality(Eligible) > 0
                         ELSE Cardinality(Eligible) = 1
          /\ stage' = "decoded"
          /\ UNCHANGED <<presence, declared, selector, candidates, bodyless, metadata,
                          ownerIsError, wirePresence>>
Next == Create \/ Encode \/ Decode
Spec == Init /\ [][Next]_vars
NamedFailure == stage # "new" => ownerIsError
ConcreteIdentity == stage \in {"encoded", "decoded"} => SelectorAgrees
PresencePreserved == stage \in {"encoded", "decoded"} => wirePresence = presence
MetadataConcrete == stage \in {"encoded", "decoded"} /\ metadata => presence = "value"
ExactlyOne == accepted => Cardinality(matches) = 1
OwnedDecode == accepted => declared \in matches
=============================================================================
