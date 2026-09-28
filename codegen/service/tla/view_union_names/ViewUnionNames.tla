-------------------------- MODULE ViewUnionNames ---------------------------
EXTENDS Naturals, TLC
CONSTANT ReserveFirst
VARIABLES names, references, count
vars == <<names, references, count>>
Types == 1..2
Init == /\ names = [t \in Types |-> ""]
        /\ references = [t \in Types |-> ""]
        /\ count = 0
Reserve(t) == /\ names[t] = ""
              /\ names' = [names EXCEPT ![t] = IF count = 0 THEN "Block" ELSE "Block2"]
              /\ count' = count + 1
              /\ UNCHANGED references
Reference(t) == /\ references[t] = ""
                /\ (~ReserveFirst \/ names[t] # "")
                /\ references' = [references EXCEPT ![t] = IF names[t] = "" THEN "Block" ELSE names[t]]
                /\ UNCHANGED <<names, count>>
Done == \A t \in Types: names[t] # "" /\ references[t] # ""
Next == (\E t \in Types: Reserve(t) \/ Reference(t)) \/ (Done /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
ReferencesMatchDeclarations == \A t \in Types:
    (names[t] # "" /\ references[t] # "") => names[t] = references[t]
DistinctDeclarations == names[1] # "" /\ names[2] # "" => names[1] # names[2]
Terminates == <>Done
=============================================================================
