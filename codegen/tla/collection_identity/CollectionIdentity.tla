----------------------- MODULE CollectionIdentity -----------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
Types == {"first", "second", "external"}
Names == {"Item", "Item2", "Item3", "Item4"}
Name(i) == CASE i = 1 -> "Item"
            [] i = 2 -> "Item2"
            [] i = 3 -> "Item3"
            [] OTHER -> "Item4"
VARIABLES used, allocated, declared, referenced
vars == <<used, allocated, declared, referenced>>
Empty == [t \in Types |-> ""]
Init == /\ used \in {{}, {"Item"}}
        /\ allocated = Empty
        /\ declared = Empty
        /\ referenced = Empty
Allocate(t) ==
    /\ allocated[t] = ""
    /\ LET index == CHOOSE i \in 1..4 :
                         /\ Name(i) \notin used
                         /\ \A j \in 1..(i-1) : Name(j) \in used
           chosen == Name(index)
       IN /\ allocated' = [allocated EXCEPT ![t] = chosen]
          /\ declared' = [declared EXCEPT ![t] =
                            IF t = "external" THEN "Item" ELSE chosen]
          /\ used' = used \cup {chosen}
    /\ UNCHANGED referenced
Reference(t) ==
    /\ allocated[t] # ""
    /\ referenced[t] = ""
    /\ referenced' = [referenced EXCEPT ![t] =
          CASE Mode = "raw" -> "Item"
            [] Mode = "local-all" -> allocated[t]
            [] OTHER -> declared[t]]
    /\ UNCHANGED <<used, allocated, declared>>
Next == \E t \in Types : Allocate(t) \/ Reference(t)
Spec == Init /\ [][Next]_vars
TypeOK == /\ used \subseteq Names
          /\ allocated \in [Types -> Names \cup {""}]
          /\ declared \in [Types -> Names \cup {""}]
          /\ referenced \in [Types -> Names \cup {""}]
ReferenceIdentity == \A t \in Types : referenced[t] # "" => referenced[t] = declared[t]
ExternalIdentity == declared["external"] \in {"", "Item"}
=============================================================================
