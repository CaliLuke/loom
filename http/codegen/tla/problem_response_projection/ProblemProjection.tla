---------------------- MODULE ProblemProjection ----------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
Fields == {"code", "detail", "retry_hint"}
ServiceFields == {"name", "message", "id"}
VARIABLES bodyFields, phase, serverReads, serverSource, serverWrites, clientReads
vars == <<bodyFields, phase, serverReads, serverSource, serverWrites, clientReads>>

Init == /\ bodyFields \in SUBSET Fields
        /\ phase = "server"
        /\ serverReads = {}
        /\ serverSource = Fields
        /\ serverWrites = {}
        /\ clientReads = {}

(* Legacy scalar bodies select a public problem field directly from the
   differently shaped ServiceError. Object constructors write all fields
   even when mapping a field elsewhere pruned it from the body type. *)
Server == /\ phase = "server"
          /\ serverReads' = bodyFields
          /\ serverSource' = IF Mode = "legacy" /\ Cardinality(bodyFields) = 1
                              THEN ServiceFields ELSE Fields
          /\ serverWrites' = IF Mode = "legacy" /\ Cardinality(bodyFields) > 1
                              THEN Fields ELSE bodyFields
          /\ phase' = "client"
          /\ UNCHANGED <<bodyFields, clientReads>>

(* Client code must select the body members actually present. Other
   members come from mapped argument slots, not from nonexistent fields. *)
Client == /\ phase = "client"
          /\ clientReads' = IF Mode # "checked" /\ bodyFields # {}
                             THEN Fields ELSE bodyFields
          /\ phase' = "done"
          /\ UNCHANGED <<bodyFields, serverReads, serverSource, serverWrites>>
Next == Server \/ Client
Spec == Init /\ [][Next]_vars
Renderable == /\ serverReads \subseteq serverSource
              /\ serverWrites \subseteq bodyFields
              /\ clientReads \subseteq bodyFields
======================================================================
