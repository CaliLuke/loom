------------------------- MODULE DeclarationRoles ------------------------
EXTENDS Naturals, FiniteSets
CONSTANT AllocatedRoles
Contributions == {"error", "error-methods", "payload", "shared-payload", "shared-methods"}
Role(c) == IF c \in {"error-methods", "shared-methods"} THEN "methods" ELSE "type"
Allocated(c) == IF c = "payload" THEN "FooBar" ELSE "FooBar2"
Authored(c) == IF c = "payload" THEN "Foo-Bar" ELSE "FooBar"
Physical(c) == <<"common", Role(c), Allocated(c)>>
Key(c) == <<"common", Role(c), IF AllocatedRoles \/ c \in {"payload", "shared-payload"} THEN Allocated(c) ELSE Authored(c)>>
Declarations == {Physical(c) : c \in Contributions}
VARIABLES pending, seen, counts
vars == <<pending, seen, counts>>
Init == /\ pending = Contributions
        /\ seen = {}
        /\ counts = [d \in Declarations |-> 0]
Emit(c) == /\ c \in pending
           /\ pending' = pending \ {c}
           /\ seen' = seen \cup {Key(c)}
           /\ counts' = IF Key(c) \notin seen THEN [counts EXCEPT ![Physical(c)] = @ + 1] ELSE counts
Next == (\E c \in Contributions : Emit(c)) \/ (pending = {} /\ UNCHANGED vars)
Unique == \A d \in Declarations : counts[d] <= 1
Complete == pending = {} => (\A d \in Declarations : counts[d] = 1)
=============================================================================
