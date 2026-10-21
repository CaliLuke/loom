-------------------------- MODULE ActivityBinding --------------------------
EXTENDS Naturals, FiniteSets
CONSTANTS Names, Scopes, RebindFuture, PublishConflict
VARIABLES native, owned, registered, futures, nextID
vars == <<native, owned, registered, futures, nextID>>

Init == /\ native = {}
        /\ owned = {}
        /\ registered = {}
        /\ futures = {}
        /\ nextID = 0

\* An independently registered native activity can conflict with our registry.
NativeRegister(name) ==
    /\ native' = native \cup {name}
    /\ UNCHANGED <<owned, registered, futures, nextID>>

\* The registry lock spans native registration and local publication. A native
\* conflict must not publish a definition that workers would never execute.
\* owned records successful adapter registrations, excluding native conflicts.
\* PublishConflict models publication despite a failed native registration.
Register(name) ==
    /\ name \notin registered
    /\ (name \notin native \/ PublishConflict)
    /\ native' = native \cup {name}
    /\ owned' = IF name \notin native THEN owned \cup {name} ELSE owned
    /\ registered' = registered \cup {name}
    /\ UNCHANGED <<futures, nextID>>

Schedule(name, scope) ==
    /\ name \in registered
    /\ nextID < 2
    /\ futures' = futures \cup {[id |-> nextID, name |-> name,
                                 scheduled |-> scope, awaited |-> scope]}
    /\ nextID' = nextID + 1
    /\ UNCHANGED <<native, owned, registered>>

\* Counterexample: Get uses a later caller's workflow context instead of the
\* scope captured when the command was scheduled.
Get(future, scope) ==
    /\ future \in futures
    /\ futures' = (futures \ {future}) \cup
         {[future EXCEPT !.awaited = IF RebindFuture THEN scope ELSE @]}
    /\ UNCHANGED <<native, owned, registered, nextID>>

Next == \/ \E name \in Names: NativeRegister(name) \/ Register(name)
        \/ \E name \in Names, scope \in Scopes: Schedule(name, scope)
        \/ \E future \in futures, scope \in Scopes: Get(future, scope)

PublishedDefinitionsExist == registered \subseteq owned /\ owned \subseteq native
FutureScopeIsStable == \A future \in futures:
    future.scheduled = future.awaited /\ future.name \in registered
Spec == Init /\ [][Next]_vars
=============================================================================
