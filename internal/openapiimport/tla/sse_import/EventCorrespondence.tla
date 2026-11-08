------------------------- MODULE EventCorrespondence -------------------------
EXTENDS Naturals, Sequences, FiniteSets, TLC
CONSTANTS Mode, CaseFilter, OrderFilter
VARIABLES scenario, order, phase, eligible, selected, shared, preparations,
          responses, message, visited
vars == <<scenario, order, phase, eligible, selected, shared, preparations,
          responses, message, visited>>

Legacy == {"plain", "explicitEqual", "localExample", "primitive", "excluded",
           "selectedMember", "bodyChanged", "unknownComparison", "emptyLegacy",
           "mixed", "docs", "mapped", "projected"}
Handshakes == {"handshake", "handshakeCollision", "suppressed", "custom", "recursive",
               "handshakeMapped", "handshakeProjected", "secondDocs", "handshakeMixed"}
Cases == Legacy \cup Handshakes
Allowed == {"plain", "explicitEqual", "localExample", "primitive", "handshake",
            "handshakeCollision", "suppressed", "custom", "recursive"}
Ids(s) == IF s \in Handshakes THEN {1, 2} ELSE {1}
Orders(s) == IF s \in Handshakes THEN {<<1, 2>>, <<2, 1>>} ELSE {<<1>>}
Endpoint(s) == [mixed |-> s \in {"mixed", "handshakeMixed"},
                mapping |-> s \in {"mapped", "handshakeMapped"},
                projection |-> s \in {"projected", "handshakeProjected"},
                eventCapture |-> "event-capture", resultCapture |-> "result-capture"]
(* This is a record for EACH response, not an endpoint aggregate. *)
ResponseFacts(s) == [i \in Ids(s) |->
  [key |-> IF i = 1 THEN "200" ELSE "201",
   role |-> IF s \in Handshakes THEN "handshake" ELSE "result",
   docs |-> s = "docs" \/ (s = "secondDocs" /\ i = 2),
   body |-> s \notin Handshakes /\ s # "emptyLegacy",
   selection |-> s = "selectedMember",
   excluded |-> IF s = "excluded" THEN {"value"} ELSE {},
   comparison |-> IF s = "bodyChanged" THEN "different"
                  ELSE IF s = "unknownComparison" THEN "unsupported" ELSE "equal",
   metadataNames |-> IF s = "handshakeCollision" THEN {"value"} ELSE {}]]
ResponseWhole(f) == ~f.docs /\
  IF f.role = "handshake" THEN TRUE
  ELSE f.body /\ ~f.selection /\ f.excluded = {} /\ f.comparison = "equal"
CheckedResponse(f) ==
  IF Mode = "ignore-docs" THEN ResponseWhole([f EXCEPT !.docs = FALSE])
  ELSE IF Mode = "ignore-exclusions" THEN ResponseWhole([f EXCEPT !.excluded = {}])
  ELSE IF Mode = "ignore-body-contract" THEN ResponseWhole([f EXCEPT !.comparison = "equal"])
  ELSE IF Mode = "empty-handshake" THEN ResponseWhole(f) /\ f.body
  ELSE IF Mode = "name-collision" THEN ResponseWhole(f) /\ f.metadataNames = {}
  ELSE ResponseWhole(f)
Choose(s,o) ==
  LET e == Endpoint(s)
      rs == ResponseFacts(s)
      each == IF Mode = "first-status-only" THEN CheckedResponse(rs[o[1]])
              ELSE \A i \in Ids(s): CheckedResponse(rs[i])
  IN (Mode = "ignore-mixed" \/ ~e.mixed) /\ ~e.mapping /\ ~e.projection /\ each /\
     (Mode # "equal-capture" \/ e.eventCapture = e.resultCapture)

Project(s,v) == IF s = "suppressed" THEN <<>>
               ELSE IF s = "custom" THEN <<"projected", v>> ELSE <<v>>
Empty == [owner |-> "none", root |-> <<>>,
          child |-> [owner |-> "none", value |-> <<>>], cut |-> "none"]
EventOutcome(s) ==
  [owner |-> "event-capture",
   root |-> Project(s, IF s = "localExample" THEN "authored-local"
                      ELSE IF s = "primitive" THEN "primitive-value"
                      ELSE IF s = "handshakeMapped" THEN "mapped-event-sample"
                      ELSE IF s = "handshakeProjected" THEN "projected-event-sample"
                      ELSE "event-sample"),
   child |-> [owner |-> "declaration-child", value |-> Project(s,"child-sample")],
   cut |-> IF s = "recursive" THEN "event-cut" ELSE "none"]
ResponseOutcome(i) ==
  [owner |-> IF i = 1 THEN "response-200" ELSE "response-201",
   root |-> <<IF i = 1 THEN "sample-200" ELSE "sample-201">>,
   child |-> [owner |-> "declaration-child", value |-> <<"child-sample">>], cut |-> "none"]
ExistingResponse(s,i) ==
  [owner |-> "existing-response", root |-> <<s, IF i = 1 THEN "200" ELSE "201">>,
   child |-> [owner |-> "existing-child", value |-> <<"sample">>], cut |-> "none"]
DocsOutcome(s,i) ==
  [owner |-> "independent-docs", root |-> <<s, IF i = 1 THEN "docs-200" ELSE "docs-201">>,
   child |-> [owner |-> "docs-child", value |-> <<"authored-doc-sample">>], cut |-> "none"]
ExistingMessage(s) ==
  [owner |-> "existing-message", root |-> <<s>>,
   child |-> [owner |-> "existing-child", value |-> <<"sample">>], cut |-> "none"]
IndependentAsync(s) == [EventOutcome(s) EXCEPT !.owner = "async-baseline", !.root = <<"async-sample">>]
NeedEvent == scenario \in Handshakes \/ eligible
ExpectedPayload(s,i) ==
  IF s \in Handshakes THEN IF ResponseFacts(s)[i].docs THEN DocsOutcome(s,i) ELSE EventOutcome(s)
  ELSE IF s \in Allowed THEN EventOutcome(s) ELSE ExistingResponse(s,i)

Init == /\ scenario \in IF CaseFilter = "all" THEN Cases ELSE {CaseFilter}
        /\ order \in {o \in Orders(scenario): OrderFilter = "all" \/
                     (OrderFilter = "forward" /\ o[1] = 1) \/
                     (OrderFilter = "reverse" /\ o[1] = 2)}
        /\ phase = "select" /\ eligible = FALSE /\ selected = "none" /\ shared = Empty
        /\ preparations = 0 /\ responses = [i \in Ids(scenario) |-> Empty]
        /\ message = Empty /\ visited = {}
Select == /\ phase = "select"
          /\ eligible' = Choose(scenario,order)
          /\ selected' = IF scenario \in Handshakes \/ Choose(scenario,order)
                         THEN IF Mode = "wrong-capture" THEN Endpoint(scenario).resultCapture
                              ELSE Endpoint(scenario).eventCapture ELSE "none"
          /\ phase' = "prepare"
          /\ UNCHANGED <<scenario,order,shared,preparations,responses,message,visited>>
Prepare == /\ phase = "prepare"
           /\ LET prepare == IF Mode = "eligible-only-prepare" THEN eligible ELSE NeedEvent
              IN /\ shared' = IF prepare THEN [EventOutcome(scenario) EXCEPT !.owner = selected] ELSE Empty
                 /\ preparations' = IF prepare THEN 1 ELSE 0
           /\ phase' = "responses"
           /\ UNCHANGED <<scenario,order,eligible,selected,responses,message,visited>>
Visit == /\ phase = "responses" /\ visited # Ids(scenario)
         /\ LET i == IF order[1] \notin visited THEN order[1] ELSE order[2]
                f == ResponseFacts(scenario)[i]
                payload == IF scenario \in Handshakes /\ f.docs /\ Mode # "erase-docs"
                           THEN DocsOutcome(scenario,i)
                           ELSE IF NeedEvent THEN
                             IF Mode \in {"current-algorithm","per-status","reprepare-status"} THEN ResponseOutcome(i)
                             ELSE IF Mode = "first-visited" /\ visited = {} THEN ResponseOutcome(i)
                             ELSE shared
                           ELSE ExistingResponse(scenario,i)
            IN /\ responses' = [responses EXCEPT ![i] = payload]
               /\ shared' = IF Mode = "first-visited" /\ visited = {} THEN payload ELSE shared
               /\ preparations' = IF Mode = "reprepare-status" THEN preparations + 1 ELSE preparations
               /\ visited' = visited \cup {i}
         /\ UNCHANGED <<scenario,order,phase,eligible,selected,message>>
Derive == /\ phase = "responses" /\ visited = Ids(scenario)
          /\ message' = IF ~eligible THEN IF Mode = "overwrite-ineligible" THEN EventOutcome(scenario) ELSE ExistingMessage(scenario)
                         ELSE IF Mode \in {"current-algorithm","async-only"} THEN IndependentAsync(scenario) ELSE shared
          /\ phase' = "done"
          /\ UNCHANGED <<scenario,order,eligible,selected,shared,preparations,responses,visited>>
Next == Select \/ Prepare \/ Visit \/ Derive
Spec == Init /\ [][Next]_vars

ModelDomain == /\ scenario \in Cases /\ order \in Orders(scenario)
               /\ \A i \in Ids(scenario):
                    (Cardinality(Ids(scenario)) > 1 => ResponseFacts(scenario)[i].role = "handshake")
               /\ (scenario = "secondDocs" => ~ResponseFacts(scenario)[1].docs /\ ResponseFacts(scenario)[2].docs)
BranchSound == phase = "done" /\ eligible => scenario \in Allowed
BranchComplete == phase = "done" /\ scenario \in Allowed => eligible
EventAuthority == phase = "done" =>
  IF scenario \in Handshakes \cup Allowed
  THEN selected = Endpoint(scenario).eventCapture /\ shared = EventOutcome(scenario) /\ preparations = 1
  ELSE shared = Empty /\ preparations = 0
PayloadAuthority == phase = "done" => \A i \in Ids(scenario): responses[i] = ExpectedPayload(scenario,i)
MessageAuthority == phase = "done" => message = IF scenario \in Allowed THEN EventOutcome(scenario) ELSE ExistingMessage(scenario)
OldPaths == phase = "done" /\ scenario \in Legacy \ Allowed =>
              responses = [i \in Ids(scenario) |-> ExistingResponse(scenario,i)] /\ message = ExistingMessage(scenario)
NoEligible == ~(phase = "done" /\ eligible)
NoHandshake == ~(phase = "done" /\ scenario = "handshakeCollision" /\ eligible /\ preparations = 1)
NoMapped == ~(phase = "done" /\ scenario = "handshakeMapped" /\ ~eligible /\
              responses[1] = EventOutcome(scenario) /\ responses[2] = responses[1] /\ message = ExistingMessage(scenario))
NoSecondDocs == ~(phase = "done" /\ scenario = "secondDocs" /\ ~eligible /\
                 responses[1] = EventOutcome(scenario) /\ responses[2] = DocsOutcome(scenario,2) /\ message = ExistingMessage(scenario))
NoMixed == ~(phase = "done" /\ scenario = "mixed" /\ ~eligible /\ message = ExistingMessage(scenario))
NoDocs == ~(phase = "done" /\ scenario = "docs" /\ ~eligible /\ message = ExistingMessage(scenario))
NoReverse == ~(phase = "done" /\ scenario = "handshake" /\ order = <<2,1>> /\ responses[1] = responses[2])
NoSuppressed == ~(phase = "done" /\ scenario = "suppressed" /\ shared.root = <<>>)
NoRecursive == ~(phase = "done" /\ scenario = "recursive" /\ shared.cut = "event-cut")
=============================================================================
