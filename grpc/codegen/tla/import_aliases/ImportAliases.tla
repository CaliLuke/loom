-------------------------- MODULE ImportAliases --------------------------
EXTENDS Naturals, FiniteSets, Sequences
CONSTANTS Services, Legacy, ReserveLocals
VARIABLES order, phase, position, protobuf, service, used, locals

vars == <<order, phase, position, protobuf, service, used, locals>>
Reserved == {"loom", "loompb", "protojson", "context"}

ProtobufChoices(name) ==
  CASE name = "loom" -> <<"loompb", "loompb2", "loompb3">>
    [] name = "loompb" -> <<"loompbpb", "loompbpb2", "loompbpb3">>
    [] name = "loompb2" -> <<"loompb2pb", "loompb2pb2", "loompb2pb3">>
    [] name = "protojson" -> <<"protojsonpb", "protojsonpb2", "protojsonpb3">>
    [] name = "protojsonsvc" -> <<"protojsonsvcpb", "protojsonsvcpb2", "protojsonsvcpb3">>
    [] name = "protojsonsvc2" -> <<"protojsonsvc2pb", "protojsonsvc2pb2", "protojsonsvc2pb3">>

ServiceChoices(name) ==
  CASE name = "loom" -> <<"loom", "loomsvc", "loomsvc2", "loomsvc3", "loomsvc4">>
    [] name = "loompb" -> <<"loompb", "loompbsvc", "loompbsvc2", "loompbsvc3", "loompbsvc4">>
    [] name = "loompb2" -> <<"loompb2", "loompb2svc", "loompb2svc2", "loompb2svc3", "loompb2svc4">>
    [] name = "protojson" -> <<"protojson", "protojsonsvc", "protojsonsvc2", "protojsonsvc3", "protojsonsvc4">>
    [] name = "protojsonsvc" -> <<"protojsonsvc", "protojsonsvcsvc", "protojsonsvcsvc2", "protojsonsvcsvc3", "protojsonsvcsvc4">>
    [] name = "protojsonsvc2" -> <<"protojsonsvc2", "protojsonsvc2svc", "protojsonsvc2svc2", "protojsonsvc2svc3", "protojsonsvc2svc4">>

FirstFreeFrom(choices, occupied) ==
  choices[CHOOSE i \in 1..Len(choices):
    choices[i] \notin occupied /\ \A j \in 1..(i-1): choices[j] \in occupied]

Orders == {p \in [1..Cardinality(Services) -> Services]:
  \A i, j \in DOMAIN p: i # j => p[i] # p[j]}

Init == /\ order \in Orders
        /\ phase = "protobuf"
        /\ position = 1
        /\ protobuf = [s \in Services |-> ""]
        /\ service = [s \in Services |-> ""]
        /\ used = Reserved
        /\ locals = [s \in Services |-> <<>>]

AllocateProtobuf ==
  /\ phase = "protobuf"
  /\ position <= Cardinality(Services)
  /\ LET name == order[position]
          choices == ProtobufChoices(name)
          alias == IF Legacy THEN choices[1] ELSE FirstFreeFrom(choices, used)
      IN /\ protobuf' = [protobuf EXCEPT ![name] = alias]
         /\ used' = used \cup {alias}
  /\ position' = position + 1
  /\ UNCHANGED <<order, phase, service, locals>>

BeginServices ==
  /\ phase = "protobuf"
  /\ position > Cardinality(Services)
  /\ phase' = "service"
  /\ position' = 1
  /\ UNCHANGED <<order, protobuf, service, used, locals>>

AllocateService ==
  /\ phase = "service"
  /\ position <= Cardinality(Services)
  /\ LET name == order[position]
          choices == ServiceChoices(name)
          alias == IF Legacy THEN choices[1] ELSE FirstFreeFrom(choices, used)
      IN /\ service' = [service EXCEPT ![name] = alias]
         /\ used' = used \cup {alias}
  /\ position' = position + 1
  /\ UNCHANGED <<order, phase, protobuf, locals>>

MetadataNames == <<"protojsonsvc", "protojsonsvc2", "loompb2", "loompb22">>
LocalChoices(name) == <<name, name \o "2", name \o "3", name \o "4", name \o "5">>
LocalNames(s) == {locals[s][i]: i \in 1..Len(locals[s])}
Imports(s) == Reserved \cup {protobuf[s], service[s]}

BeginLocals ==
  /\ phase = "service"
  /\ position > Cardinality(Services)
  /\ phase' = "locals"
  /\ position' = 1
  /\ UNCHANGED <<order, protobuf, service, used, locals>>

AllocateLocals ==
  /\ phase = "locals"
  /\ position <= Len(MetadataNames)
  /\ locals' = [s \in Services |->
       Append(locals[s], FirstFreeFrom(LocalChoices(MetadataNames[position]),
         LocalNames(s) \cup IF ReserveLocals THEN Imports(s) ELSE {}))]
  /\ position' = position + 1
  /\ UNCHANGED <<order, phase, protobuf, service, used>>

Finish == /\ phase = "locals"
          /\ position > Len(MetadataNames)
          /\ phase' = "done"
          /\ UNCHANGED <<order, position, protobuf, service, used, locals>>

Next == AllocateProtobuf \/ BeginServices \/ AllocateService \/ BeginLocals \/ AllocateLocals \/ Finish
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
Allocated == ({protobuf[s]: s \in Services} \cup {service[s]: s \in Services}) \ {""}
NamespaceSafe ==
  /\ Allocated \cap Reserved = {}
  /\ Cardinality(Allocated) =
      Cardinality({s \in Services: protobuf[s] # ""}) +
      Cardinality({s \in Services: service[s] # ""})
  /\ used = Reserved \cup Allocated
LocalsSafe == \A s \in Services:
  /\ LocalNames(s) \cap Imports(s) = {}
  /\ Cardinality(LocalNames(s)) = Len(locals[s])
EventuallyDone == <> (phase = "done")
=============================================================================
