------------------------- MODULE ConstructorReference ------------------------
EXTENDS Naturals
CONSTANTS Deferred, EarlyCompletion
VARIABLES registered, resolved, refined, published, valid
vars == <<registered, resolved, refined, published, valid>>
Init == /\ registered = FALSE /\ resolved = FALSE /\ published = FALSE
        /\ refined = ~Deferred /\ valid = Deferred
Register == /\ ~registered /\ registered' = TRUE
            /\ UNCHANGED <<resolved, refined, published, valid>>
Resolve == /\ registered /\ ~resolved /\ resolved' = TRUE
           /\ UNCHANGED <<registered, refined, published, valid>>
Refine == /\ resolved /\ ~refined /\ refined' = TRUE
          /\ valid' = registered /\ UNCHANGED <<registered, resolved, published>>
Publish == /\ registered /\ ~published /\ published' = TRUE
           /\ (EarlyCompletion => resolved /\ refined)
           /\ UNCHANGED <<registered, resolved, refined, valid>>
Next == Register \/ Resolve \/ Refine \/ Publish
KnownBeforeRefinement == refined => valid
CompleteBeforePublication == published => registered /\ resolved /\ refined /\ valid
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
Terminates == <>published
=============================================================================
