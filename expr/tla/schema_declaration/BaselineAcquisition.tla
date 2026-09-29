----------------------- MODULE BaselineAcquisition -----------------------
EXTENDS Naturals, Sequences
CONSTANTS Policy, Scenario
VARIABLES scenario, authored, requests, memo, acquired
vars == <<scenario, authored, requests, memo, acquired>>

Occurrences == {"ordinary", "left", "right"}
Fields == {"event", "id", "retry"}
Absent == "absent"
DefaultValues(field) ==
    CASE field = "event" -> {Absent, "default-event", "other-event"}
      [] field = "id" -> {Absent, "default-id", "other-id"}
      [] OTHER -> {Absent, "1500", "3000"}
DefaultMaps == {d \in [Fields -> {Absent, "default-event", "other-event",
                                "default-id", "other-id", "1500", "3000"}] :
                 \A f \in Fields : d[f] \in DefaultValues(f)}
Payload(defaults) == [defaults |-> defaults, examples |-> <<"same-example">>]
Payloads == {Payload(d) : d \in DefaultMaps}
EmptyDefaults == Payload([f \in Fields |-> Absent])

\* All three occurrences have the same structural fingerprint. Public names
\* remain distinct and are never changed to encode annotation differences.
Shape(occurrence) == "same-object-shape"
PublicName(occurrence) ==
    CASE occurrence = "ordinary" -> "OptionalResponseBody"
      [] occurrence = "left" -> "DefaultedResult"
      [] OTHER -> "OtherNamedResult"
Usage(occurrence) ==
    IF occurrence = "ordinary" THEN "http-response" ELSE "async-inline"
Entry(occurrence) ==
    [owner |-> occurrence, usage |-> Usage(occurrence),
     shape |-> Shape(occurrence), payload |-> authored[occurrence]]

Init ==
    /\ scenario \in IF Scenario = "all"
                    THEN {"cross-consumer", "same-message"} ELSE {Scenario}
    /\ authored \in [Occurrences -> Payloads]
    \* No ordinary consumer participates in the isolated-message scenario.
    /\ scenario = "same-message" => authored["ordinary"] = EmptyDefaults
    /\ requests \in {<<"left", "right", "left", "right">>,
                      <<"right", "left", "right", "left">>}
    /\ memo = IF scenario = "cross-consumer"
              THEN <<Entry("ordinary")>> ELSE <<>>
    /\ acquired = <<>>

Compatible(entry, occurrence) ==
    /\ entry.shape = Shape(occurrence)
    /\ CASE Policy = "structural" -> TRUE
         [] Policy = "annotation-aware" -> entry.payload = authored[occurrence]
         [] Policy = "occurrence-owned" ->
              entry.owner = occurrence /\ entry.usage = Usage(occurrence)
         [] Policy = "never-cache" -> FALSE
         [] OTHER -> FALSE
Matches(occurrence) ==
    {i \in 1..Len(memo) : Compatible(memo[i], occurrence)}
First(indices) == CHOOSE i \in indices : \A j \in indices : i <= j

\* Reuse returns the existing payload, not a reconstruction from the request.
\* A miss materializes the authored payload and allocates a stable memo ID.
Acquire ==
    /\ Len(acquired) < Len(requests)
    /\ LET occurrence == requests[Len(acquired) + 1]
           matches == Matches(occurrence)
           id == IF matches = {} THEN Len(memo) + 1 ELSE First(matches)
           selected == IF matches = {} THEN Entry(occurrence) ELSE memo[id]
       IN /\ memo' = IF matches = {} THEN Append(memo, selected) ELSE memo
          /\ acquired' = Append(acquired,
                  [occurrence |-> occurrence, id |-> id,
                   payload |-> selected.payload])
    /\ UNCHANGED <<scenario, authored, requests>>
Spec == Init /\ [][Acquire]_vars

\* Expected annotation authority is immutable input, independent of the memo
\* and selected result. Equality includes absent and present nested defaults.
AuthoredPayloadPreserved ==
    \A i \in 1..Len(acquired) :
        acquired[i].payload = authored[acquired[i].occurrence]
\* Reject a vacuous repair that always bypasses the cache. A repeated exact
\* occurrence/use must return the same schema identity despite other requests.
RepeatedOccurrenceReused ==
    \A i, j \in 1..Len(acquired) :
        acquired[i].occurrence = acquired[j].occurrence =>
          acquired[i].id = acquired[j].id
MemoIdentityStable ==
    \A i \in 1..Len(acquired) :
        acquired[i].id \in 1..Len(memo) /\
        memo[acquired[i].id].payload = acquired[i].payload
=============================================================================
