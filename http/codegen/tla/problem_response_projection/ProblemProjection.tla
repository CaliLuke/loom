---------------------- MODULE ProblemProjection ----------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
Fields == {"code", "detail", "retry_hint"}
ServiceFields == {"name", "message", "id"}
VARIABLES bodyFields, selection, phase, serverReads, serverSource, serverWrites,
          clientReads, constructorField
vars == <<bodyFields, selection, phase, serverReads, serverSource, serverWrites,
          clientReads, constructorField>>

Init == /\ bodyFields \in SUBSET Fields
        /\ selection \in IF Cardinality(bodyFields) = 1
                           THEN {"object", "member"} ELSE {"object"}
        /\ phase = "server"
        /\ serverReads = {}
        /\ serverSource = Fields
        /\ serverWrites = {}
        /\ clientReads = {}
        /\ constructorField = ""

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
          /\ UNCHANGED <<bodyFields, selection, clientReads, constructorField>>

(* Client code must select the body members actually present. Other
   members come from mapped argument slots, not from nonexistent fields. *)
Client == /\ phase = "client"
          /\ clientReads' = IF Mode \in {"legacy", "source_only"} /\ bodyFields # {}
                             THEN Fields ELSE bodyFields
          /\ constructorField' = IF Mode = "wire_target" /\ selection = "member"
                                  THEN CHOOSE field \in bodyFields : TRUE ELSE ""
          /\ phase' = "done"
          /\ UNCHANGED <<bodyFields, selection, serverReads, serverSource, serverWrites>>
Next == Server \/ Client
Spec == Init /\ [][Next]_vars
Renderable == /\ serverReads \subseteq serverSource
              /\ serverWrites \subseteq bodyFields
              /\ clientReads \subseteq bodyFields

(* ProblemErrorFromBody has already produced a whole ServiceError. Its result
   must not be assigned to the selected public problem member a second time. *)
NativeResultNotRewrapped == constructorField = ""
======================================================================
