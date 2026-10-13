------------------------- MODULE FixedLocals -------------------------
EXTENDS Naturals, FiniteSets, Sequences
CONSTANTS ReserveFixedLocals, ReserveMetadataImports
VARIABLES order, position, phase, aliases, protobuf, used, metadata
vars == <<order, position, phase, aliases, protobuf, used, metadata>>
Services == {"md", "mdsvc", "mdsvc2", "ctx", "req"}
FixedImports == {"context", "metadata", "c"}
FixedLocals == {"ctx", "md", "p", "reqpb"}
CustomImports == {"relocated"}
ImportReservations == FixedImports \cup
  IF ReserveFixedLocals THEN FixedLocals ELSE {}
Choices(n) == <<n, n \o "svc", n \o "svc2", n \o "svc3", n \o "svc4">>
LocalChoices(n) == <<n, n \o "2", n \o "3", n \o "4", n \o "5">>
FirstFree(choices, occupied) == choices[CHOOSE i \in 1..Len(choices):
  choices[i] \notin occupied /\ \A j \in 1..(i-1): choices[j] \in occupied]
Orders == {p \in [1..Cardinality(Services) -> Services]:
  \A i, j \in DOMAIN p: i # j => p[i] # p[j]}
Allocated == ({aliases[s]: s \in Services} \cup {protobuf[s]: s \in Services}) \ {""}
MetadataNames == <<"mdsvc", "mdsvc2", "relocated", "reqpb2", "ctx", "p">>
LocalNames(s) == {metadata[s][i]: i \in 1..Len(metadata[s])}
ActualImports(s) == FixedImports \cup CustomImports \cup {aliases[s], protobuf[s]}
LocalReservations(s) == FixedImports \cup FixedLocals \cup
  IF ReserveMetadataImports THEN ActualImports(s) ELSE {}
Init == /\ order \in Orders
        /\ position = 1
        /\ phase = "protobuf"
        /\ aliases = [s \in Services |-> ""]
        /\ protobuf = [s \in Services |-> ""]
        /\ used = ImportReservations
        /\ metadata = [s \in Services |-> <<>>]
AllocateProtobuf ==
  /\ phase = "protobuf"
  /\ position <= Cardinality(Services)
  /\ LET s == order[position]
         alias == FirstFree(LocalChoices(s \o "pb"), used)
     IN /\ protobuf' = [protobuf EXCEPT ![s] = alias]
        /\ used' = used \cup {alias}
  /\ position' = position + 1
  /\ UNCHANGED <<order, phase, aliases, metadata>>
BeginImports ==
  /\ phase = "protobuf"
  /\ position > Cardinality(Services)
  /\ phase' = "imports"
  /\ position' = 1
  /\ UNCHANGED <<order, aliases, protobuf, used, metadata>>
AllocateImport ==
  /\ phase = "imports"
  /\ position <= Cardinality(Services)
  /\ LET s == order[position]
         alias == FirstFree(Choices(s), used)
     IN /\ aliases' = [aliases EXCEPT ![s] = alias]
        /\ used' = used \cup {alias}
  /\ position' = position + 1
  /\ UNCHANGED <<order, phase, protobuf, metadata>>
BeginMetadata ==
  /\ phase = "imports"
  /\ position > Cardinality(Services)
  /\ phase' = "metadata"
  /\ position' = 1
  /\ UNCHANGED <<order, aliases, protobuf, used, metadata>>
AllocateMetadata ==
  /\ phase = "metadata"
  /\ position <= Len(MetadataNames)
  /\ metadata' = [s \in Services |-> Append(metadata[s],
      FirstFree(LocalChoices(MetadataNames[position]),
                LocalReservations(s) \cup LocalNames(s)))]
  /\ position' = position + 1
  /\ UNCHANGED <<order, phase, aliases, protobuf, used>>
Finish == /\ phase = "metadata"
          /\ position > Len(MetadataNames)
          /\ phase' = "done"
          /\ UNCHANGED <<order, position, aliases, protobuf, used, metadata>>
Next == AllocateProtobuf \/ BeginImports \/ AllocateImport \/ BeginMetadata \/ AllocateMetadata \/ Finish
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
ImportsVisible ==
  /\ Allocated \cap (FixedImports \cup FixedLocals) = {}
  /\ Cardinality(Allocated) = Cardinality({s \in Services: aliases[s] # ""}) +
     Cardinality({s \in Services: protobuf[s] # ""})
  /\ used = ImportReservations \cup Allocated
MetadataSafe == \A s \in Services:
  /\ LocalNames(s) \cap (ActualImports(s) \cup FixedLocals) = {}
  /\ Cardinality(LocalNames(s)) = Len(metadata[s])
EventuallyDone == <> (phase = "done")
=============================================================================
