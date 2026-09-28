-------------------------- MODULE SecurityBindings --------------------------
EXTENDS Naturals, FiniteSets, Sequences
CONSTANT Mode
Bindings == {"a", "b", "c"}
Order == <<"a", "b", "c">>
Names == 1..6
Base(b) == IF b = "c" THEN 2 ELSE 1
Candidate(b) == 2
Reserved == {Base(b): b \in Bindings}
VARIABLES allocated, emitted, names, definitions
vars == <<allocated, emitted, names, definitions>>
Init == /\ allocated = 0
        /\ emitted = {}
        /\ names = [b \in Bindings |-> 0]
        /\ definitions = [n \in Names |-> "none"]
Used == {names[Order[i]]: i \in 1..allocated}
Available(b) == {n \in Names: n >= Candidate(b) /\ n \notin Used /\ (Mode # "checked" \/ n \notin Reserved)}
Minimum(ns) == CHOOSE n \in ns: \A m \in ns: n <= m
Allocate == /\ allocated < Len(Order)
            /\ LET b == Order[allocated + 1] IN
                names' = [names EXCEPT ![b] = IF Mode = "legacy" \/ b = "c" THEN Base(b) ELSE Minimum(Available(b))]
            /\ allocated' = allocated + 1
            /\ UNCHANGED <<emitted, definitions>>
Emit(b) == /\ allocated = Len(Order)
           /\ b \notin emitted
           /\ definitions' = [definitions EXCEPT ![names[b]] = b]
           /\ emitted' = emitted \cup {b}
           /\ UNCHANGED <<allocated, names>>
Done == emitted = Bindings
Next == Allocate \/ (\E b \in Bindings: Emit(b)) \/ (Done /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
BindingCorrect == \A b \in emitted: definitions[names[b]] = b
UniqueBindings == allocated = Len(Order) => \A a,b \in Bindings: names[a] = names[b] => a = b
CanonicalNames == allocated = Len(Order) => names = [b \in Bindings |-> CASE b = "a" -> 3 [] b = "b" -> 4 [] OTHER -> 2]
Terminates == <>Done
=============================================================================
