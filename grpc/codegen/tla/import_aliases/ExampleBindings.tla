----------------------- MODULE ExampleBindings -----------------------
EXTENDS Naturals, FiniteSets, Sequences
CONSTANT RetainBindings
VARIABLES position, used, bindings, references
vars == <<position, used, bindings, references>>
\* File roles occur in production's server/service/protobuf allocation order.
\* The first service's escaped transport alias is psvc, while an authored
\* second service already requests psvcsvr.
Requested == <<"psvcsvr", "psvc", "ppb", "psvcsvrsvr", "psvcsvr", "psvcsvrpb">>
Reserved == {"context", "grpc", "net", "ctx"}
Choices(n) == <<n, n \o "2", n \o "3">>
FirstFree(c) == c[CHOOSE i \in 1..Len(c):
  c[i] \notin used /\ \A j \in 1..(i-1): c[j] \in used]
Init == /\ position = 1
        /\ used = Reserved
        /\ bindings = <<>>
        /\ references = <<>>
Allocate ==
 /\ position <= Len(Requested)
 /\ LET chosen == FirstFree(Choices(Requested[position]))
    IN /\ used' = used \cup {chosen}
       /\ bindings' = Append(bindings, chosen)
       /\ references' = Append(references,
          IF RetainBindings THEN chosen ELSE Requested[position])
 /\ position' = position + 1
Next == Allocate
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
HeaderUnique == Cardinality({bindings[i]: i \in 1..Len(bindings)}) = Len(bindings)
ReferencesAgree == bindings = references
EventuallyDone == <> (position > Len(Requested))
=============================================================================
