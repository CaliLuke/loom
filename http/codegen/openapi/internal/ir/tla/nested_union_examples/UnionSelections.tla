---------------------- MODULE UnionSelections ----------------------
EXTENDS Naturals, Sequences, FiniteSets
CONSTANT Mode
Occurrences == {1,2}
OuterTags == {"s","t"}
InnerTags == {"Leaf","Other"}
Paths == OuterTags \X InnerTags
VARIABLES chosen, remaining, emitted, cache
vars == <<chosen, remaining, emitted, cache>>
Init == /\ chosen \in [Occurrences -> Paths]
        /\ remaining = Occurrences
        /\ emitted = [n \in Occurrences |-> <<>>]
        /\ cache = [tag \in InnerTags |-> <<>>]
Emit(n) == LET path == chosen[n]
               payload == <<path[2]>>
               wire == CASE Mode = "legacy" -> payload
                         [] Mode = "payload-cache" -> IF cache[path[2]] = <<>> THEN path ELSE cache[path[2]]
                         [] OTHER -> path
           IN /\ n \in remaining
              /\ emitted' = [emitted EXCEPT ![n] = wire]
              /\ cache' = [cache EXCEPT ![path[2]] = wire]
              /\ remaining' = remaining \ {n}
              /\ UNCHANGED chosen
Next == (\E n \in remaining: Emit(n)) \/ (remaining = {} /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
SelectionsPreserved == \A n \in Occurrences \ remaining: emitted[n] = chosen[n]
Terminates == <>(remaining = {})
====================================================================
