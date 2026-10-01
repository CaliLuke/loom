--------------------------- MODULE Identifiers ---------------------------
EXTENDS Naturals, FiniteSets, Sequences
CONSTANT Legacy, RecomputeReferences
VARIABLES order, position, declarations, references, used

Requests == {"enClient", "en2Client", "commandUsage", "methodUsage",
             "commandFlags", "methodFlags"}
Reserved == {"enc", "ParseEndpoint", "UsageCommands", "UsageExamples"}
Preferred(request) ==
  CASE request = "enClient" -> "enc"
    [] request = "en2Client" -> "enc2"
    [] request = "commandUsage" -> "ckRawUsage"
    [] request = "methodUsage" -> "ckRawUsage"
    [] request = "commandFlags" -> "ckRawFlags"
    [] request = "methodFlags" -> "ckRawFlags"

Suffixes == <<"", "2", "3", "4", "5", "6", "7">>
Choices(request) == [i \in 1..Len(Suffixes) |-> Preferred(request) \o Suffixes[i]]
FirstFree(request) ==
  LET choices == Choices(request)
  IN choices[CHOOSE i \in DOMAIN choices:
       choices[i] \notin used /\ \A j \in 1..(i-1): choices[j] \in used]

Orders == {p \in [1..Cardinality(Requests) -> Requests]:
  \A i, j \in DOMAIN p: i # j => p[i] # p[j]}
vars == <<order, position, declarations, references, used>>
Init == /\ order \in Orders
        /\ position = 1
        /\ declarations = [r \in Requests |-> ""]
        /\ references = [r \in Requests |-> ""]
        /\ used = Reserved

Allocate ==
  /\ position <= Cardinality(Requests)
  /\ LET request == order[position]
         name == IF Legacy THEN Preferred(request) ELSE FirstFree(request)
     IN /\ declarations' = [declarations EXCEPT ![request] = name]
        /\ references' = [references EXCEPT ![request] =
             IF RecomputeReferences THEN Preferred(request) ELSE name]
        /\ used' = used \cup {name}
  /\ position' = position + 1
  /\ UNCHANGED order

Spec == Init /\ [][Allocate]_vars /\ WF_vars(Allocate)
Allocated == {r \in Requests: declarations[r] # ""}
Names == {declarations[r]: r \in Allocated}
NamespaceSafe == /\ Names \cap Reserved = {}
                 /\ Cardinality(Names) = Cardinality(Allocated)
                 /\ used = Reserved \cup Names
ReferencesAgree == \A r \in Allocated: references[r] = declarations[r]
EventuallyDone == <> (position > Cardinality(Requests))
=============================================================================
