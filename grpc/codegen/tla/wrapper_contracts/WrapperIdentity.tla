---------------------- MODULE WrapperIdentity ----------------------
EXTENDS Naturals, FiniteSets, TLC
CONSTANT Mode
Sites == {"free", "strict", "strict-copy"}
Bound(s) == IF s = "free" THEN 0 ELSE 2
Key(s) == CASE Mode = "contract" -> IF Bound(s) = 0 THEN "free-contract" ELSE "strict-contract"
               [] Mode = "positional" -> s
               [] OTHER -> "wire"
VARIABLES remaining, owners, seenKeys, helpers, visited
vars == <<remaining, owners, seenKeys, helpers, visited>>
Init == /\ remaining = Sites
        /\ owners = [k \in ({"free-contract", "strict-contract", "wire"} \cup Sites) |-> "none"]
        /\ seenKeys = {}
        /\ helpers = {}
        /\ visited = {}
Visit(s) == /\ s \in remaining
            /\ owners' = IF Key(s) \notin seenKeys THEN [owners EXCEPT ![Key(s)] = s]
                          ELSE IF Mode = "merged" /\ Bound(s) > Bound(owners[Key(s)])
                               THEN [owners EXCEPT ![Key(s)] = s] ELSE owners
            /\ helpers' = IF (Key(s) \notin seenKeys \/ Mode = "merged") /\ (Mode = "current" \/ Bound(s) > 0)
                           THEN helpers \cup {Key(s)} ELSE helpers
            /\ seenKeys' = seenKeys \cup {Key(s)}
            /\ visited' = visited \cup {s}
            /\ remaining' = remaining \ {s}
Next == \E s \in Sites: Visit(s)
Spec == Init /\ [][Next]_vars
HelperClosure == \A s \in visited: Bound(s) > 0 => Key(s) \in helpers
PredicateFaithful == \A s \in visited: \A n \in {1, 2}:
    (n >= Bound(owners[Key(s)])) = (n >= Bound(s))
EqualContractsShare == \A a,b \in visited: Bound(a) = Bound(b) => Key(a) = Key(b)
=============================================================================
