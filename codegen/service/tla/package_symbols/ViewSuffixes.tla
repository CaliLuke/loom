-------------------------- MODULE ViewSuffixes ----------------------------
EXTENDS Naturals, Sequences, FiniteSets
CONSTANT OwnedAliases
Branches == {"tiny-view", "tiny_view", "tiny_view2"}
VARIABLES primary, pending, deferred, used, aliases, phase
vars == <<primary, pending, deferred, used, aliases, phase>>
Common(n) == {}
Preferred(b) == IF b = "tiny_view2" THEN "TinyView2" ELSE "TinyView"
Candidates(b) == <<Preferred(b), Preferred(b) \o "2", Preferred(b) \o "3",
                   Preferred(b) \o "4", Preferred(b) \o "5", Preferred(b) \o "6">>
Init == /\ primary = "View"
        /\ pending = Branches /\ deferred = {} /\ used = Common(primary)
        /\ aliases = [b \in Branches |-> ""] /\ phase = "natural"
Natural(b) ==
    /\ phase = "natural" /\ b \in pending
    /\ pending' = pending \ {b}
    /\ IF OwnedAliases /\ Preferred(b) \in used
          THEN /\ deferred' = deferred \cup {b}
               /\ UNCHANGED <<used, aliases>>
          ELSE /\ used' = used \cup {Preferred(b)}
               /\ aliases' = [aliases EXCEPT ![b] = Preferred(b)]
               /\ UNCHANGED deferred
    /\ UNCHANGED <<primary, phase>>
Suffix(b) ==
    LET candidates == Candidates(b)
        available == {i \in 1..Len(candidates) : candidates[i] \notin used}
        first == CHOOSE i \in available : \A j \in available : i <= j
        selected == candidates[first]
    IN /\ phase = "suffix" /\ b \in deferred
       /\ deferred' = deferred \ {b} /\ used' = used \cup {selected}
       /\ aliases' = [aliases EXCEPT ![b] = selected]
       /\ UNCHANGED <<primary, pending, phase>>
Next == (\E b \in pending : Natural(b)) \/ (\E b \in deferred : Suffix(b))
        \/ (phase = "natural" /\ pending = {} /\ phase' = "suffix"
             /\ UNCHANGED <<primary, pending, deferred, used, aliases>>)
        \/ (phase = "suffix" /\ deferred = {} /\ UNCHANGED vars)
Assigned == {b \in Branches : aliases[b] # ""}
IntrinsicallyDistinct == /\ \A b \in Assigned : aliases[b] \notin Common(primary)
                         /\ \A a,b \in Assigned : a # b => aliases[a] # aliases[b]
NaturalAliasesPreserved == \A b \in Assigned :
    (\A other \in Branches \ {b} : Preferred(other) # Preferred(b)) => aliases[b] = Preferred(b)
=============================================================================
