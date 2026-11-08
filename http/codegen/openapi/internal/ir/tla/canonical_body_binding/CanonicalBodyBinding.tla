------------------------ MODULE CanonicalBodyBinding ------------------------
EXTENDS Naturals, Sequences, FiniteSets
CONSTANT Mode
VARIABLES order, position, phase, captured, binding, graph, valid,
          globalWire, globalPlan, materializedPlan, refs, graphs, bindings, media, overlays, checked
vars == <<order, position, phase, captured, binding, graph, valid,
          globalWire, globalPlan, materializedPlan, refs, graphs, bindings, media, overlays, checked>>
Methods == {"first", "second", "implicit", "binary", "fresh", "conflict"}
Orders == {<<"first", "second", "implicit", "binary", "fresh", "conflict">>,
           <<"second", "first", "implicit", "binary", "fresh", "conflict">>}
Claim(m) == m \in {"first", "second", "binary", "conflict"}
Ordinary(m) == m \in {"first", "second", "implicit"}
Shape(m) == IF m = "conflict" THEN "incompatible-authored-shape" ELSE "RT-contract"
Wire(m) == IF m = "binary" THEN "binary" ELSE "ordinary-json"
Plan(m) == <<"captured-plan", m>>
PreparedBinding(m) == <<"prepared-component", m>>
CanonicalBinding == <<"recorded-canonical", "RT">>
FreshBinding == <<"fresh-allocated", "Alias_1">>
ExpectedBinding(m) == IF m = "fresh" THEN FreshBinding ELSE CanonicalBinding
Media(m) == <<"media-root-example", m>>
Overlay(m) == IF m = "first" THEN "authored-reference-local-title" ELSE "none"
CanonicalGraph == [shape |-> "RT-contract", wire |-> "ordinary-json", annotation |-> CanonicalBinding]
EmptyGraph == [shape |-> "none", wire |-> "none", annotation |-> <<"none", "none">>]
Input(m) == [method |-> m, claim |-> Claim(m), shape |-> Shape(m), wire |-> Wire(m),
             plan |-> Plan(m), prepared |-> PreparedBinding(m), media |-> Media(m), overlay |-> Overlay(m)]

Init == /\ order \in Orders /\ position = 1 /\ phase = "Reserve"
        /\ captured = Input("first") /\ binding = <<"none", "none">>
        /\ graph = EmptyGraph /\ valid = FALSE
        /\ globalWire = "ambient-wire" /\ globalPlan = <<"ambient-plan", "unrelated">> /\ materializedPlan = <<"none", "none">>
        /\ refs = [m \in Methods |-> "none"]
        /\ graphs = [m \in Methods |-> EmptyGraph]
        /\ bindings = [m \in Methods |-> <<"none", "none">>]
        /\ media = [m \in Methods |-> "none"] /\ overlays = [m \in Methods |-> "none"]
        /\ checked = {}

Reserve ==
 /\ phase = "Reserve" /\ position <= Len(order)
 /\ LET m == order[position]
        reused == IF m = "implicit" THEN bindings[order[1]] ELSE CanonicalBinding
        selected == IF m = "fresh" THEN (IF Mode = "WrongFresh" THEN CanonicalBinding ELSE FreshBinding)
                    ELSE IF Mode \in {"Current", "SortFirst"} /\ m \in {"first", "second"}
                         THEN PreparedBinding(m) ELSE reused
    IN /\ captured' = Input(m) /\ binding' = selected
 /\ graph' = EmptyGraph /\ valid' = FALSE /\ phase' = "Materialize"
 /\ UNCHANGED <<order, position, globalWire, globalPlan, materializedPlan, refs, graphs, bindings, media, overlays, checked>>

Materialize ==
 /\ phase = "Materialize"
 /\ graph' = [shape |-> captured.shape,
              wire |-> IF Mode = "DeferredGlobal" THEN globalWire ELSE captured.wire,
              annotation |-> binding]
 /\ materializedPlan' = IF Mode = "DeferredGlobal" THEN globalPlan ELSE captured.plan
 /\ phase' = IF Mode = "EarlyCommit" THEN "Commit" ELSE "Validate"
 /\ UNCHANGED <<order, position, captured, binding, valid, globalWire, globalPlan, refs, graphs, bindings, media, overlays, checked>>

Validate ==
 /\ phase = "Validate"
 (* Compatibility is an independent fixture fact; an explicit foreign shape
    cannot become equivalent through chosen naming or annotation reuse. *)
 /\ valid' = (captured.shape = "RT-contract" \/ Mode = "IgnoreConflict")
 /\ checked' = checked \cup {captured.method}
 /\ phase' = "Commit"
 /\ UNCHANGED <<order, position, captured, binding, graph, globalWire, globalPlan, materializedPlan, refs, graphs, bindings, media, overlays>>

Commit ==
 /\ phase = "Commit"
 /\ LET m == captured.method
        reference == IF ~valid /\ Mode # "EarlyCommit" THEN "rejected"
                     ELSE IF m = "fresh" THEN "Alias_1"
                     ELSE IF Mode = "ForceName" THEN "RT"
                     ELSE IF Mode = "SortFirst" /\ position = 1 THEN "RT"
                     ELSE IF graph = CanonicalGraph THEN "RT"
                     ELSE <<"RT-variant", m>>
    IN /\ refs' = [refs EXCEPT ![m] = reference]
       /\ graphs' = [graphs EXCEPT ![m] = graph]
       /\ bindings' = [bindings EXCEPT ![m] = binding]
       /\ media' = [media EXCEPT ![m] = IF Mode = "OverwriteMedia" THEN binding ELSE captured.media]
       /\ overlays' = [overlays EXCEPT ![m] = IF Mode = "DropOverlay" THEN "none" ELSE captured.overlay]
 /\ position' = position + 1
 /\ phase' = IF position = Len(order) THEN "Done" ELSE "Reserve"
 /\ UNCHANGED <<order, captured, binding, graph, valid, globalWire, globalPlan, materializedPlan, checked>>
Done == /\ phase = "Done" /\ UNCHANGED vars
Next == Reserve \/ Materialize \/ Validate \/ Commit \/ Done
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
Committed == {m \in Methods: refs[m] # "none"}
Accepted == {m \in Committed: refs[m] # "rejected"}
TypeOK == /\ position \in 1..(Len(order)+1) /\ checked \subseteq Methods
          /\ DOMAIN refs = Methods /\ DOMAIN graphs = Methods /\ DOMAIN bindings = Methods
CanonicalOrdinary == \A m \in Committed: Ordinary(m) => refs[m] = "RT"
SelectedBindingPreserved == \A m \in Accepted: bindings[m] = ExpectedBinding(m)
MediaRootPreserved == \A m \in Committed: media[m] = Media(m)
AuthoredOverlayPreserved == \A m \in Committed: overlays[m] = Overlay(m)
EncodingDistinct == "binary" \in Committed => refs["binary"] # "RT" /\ graphs["binary"].wire = "binary"
AuthoredConflictRejected == "conflict" \in Committed => refs["conflict"] = "rejected"
CanonicalGraphPreserved == \A m \in Accepted: refs[m] = "RT" => graphs[m] = CanonicalGraph
CapturedPlanPreserved == phase \in {"Validate", "Commit"} => graph.wire = captured.wire /\ materializedPlan = captured.plan
CompleteValidationBeforeCommit == Accepted \subseteq checked
Complete == <>(phase = "Done")
=============================================================================
