----------------------------- MODULE SSEImport -----------------------------
EXTENDS Naturals, Sequences, FiniteSets, TLC
CONSTANTS Mode, CaseFilter
VARIABLES scenario, order, phase, loaded, retained, closure, visited,
          diagnostics, admitted, plan, ordinary, derived
vars == <<scenario, order, phase, loaded, retained, closure, visited,
          diagnostics, admitted, plan, ordinary, derived>>

Cases == {"exact201", "class2XX", "twoEqual", "equalOwnerValues", "suppressed",
          "custom", "filterAll", "recursive", "differentExample", "differentDefault",
          "differentTarget", "differentAssertion", "declarationContent",
          "nestedContent", "badEnvelope", "oldDialect", "staleExtension",
          "outsideExtension", "wholeSchema", "projection", "mapped"}
Two(s) == s \in {"twoEqual", "equalOwnerValues", "differentExample",
                 "differentDefault", "differentTarget", "differentAssertion"}
HasExtension(s) == s \in {"recursive", "staleExtension", "outsideExtension"}
Branch(s) == s \notin {"outsideExtension", "wholeSchema"}
Allowed == {"exact201", "class2XX", "twoEqual", "equalOwnerValues",
            "suppressed", "custom", "filterAll", "recursive", "wholeSchema"}
Absent == "absent"
Project(s,v) == IF s \in {"suppressed", "filterAll"} THEN Absent
               ELSE IF s = "custom" THEN <<"projected", v>> ELSE v
RootRaw(s,i) == IF s = "differentExample" /\ i = 2 THEN "other"
                ELSE "sample"
Source(s) == [
  dialect |-> IF s = "oldDialect" THEN "3.1" ELSE "3.2",
  item |-> Branch(s), envelope |-> IF s = "badEnvelope" THEN "open" ELSE "closed",
  declarationContent |-> s \in {"declarationContent", "nestedContent"},
  mapped |-> s = "mapped", projected |-> s = "projection",
  extension |-> HasExtension(s),
  extensionMatches |-> s \notin {"staleExtension", "outsideExtension"},
  occurrences |-> [i \in 1..(IF Two(s) THEN 2 ELSE 1) |->
    [payload |-> IF s = "differentTarget" /\ i = 2 THEN "Q" ELSE "P",
     default |-> IF s = "differentDefault" /\ i = 2 THEN "d2" ELSE "d1",
     assertion |-> IF s = "differentAssertion" /\ i = 2 THEN "a2" ELSE "a1",
     raw |-> RootRaw(s,i), owner |-> IF i = 1 THEN "occurrence1" ELSE "occurrence2"]],
  child |-> [owner |-> "declaration-child", value |-> "sample"],
  cut |-> IF s = "recursive" THEN "P" ELSE "none",
  literal |-> [ref |-> "not-a-schema-edge"],
  keys |-> IF Two(s) THEN <<"200","201">>
           ELSE IF s = "class2XX" THEN <<"2XX">> ELSE <<"201">>]
Indices(s) == DOMAIN Source(s).occurrences
Semantic(o) == <<o.payload, o.default, o.assertion, o.raw>>
Equivalent(g) == \A i,j \in DOMAIN g.occurrences :
                  Semantic(g.occurrences[i]) = Semantic(g.occurrences[j])
ExpectedEdges(s) == IF ~Branch(s) THEN {}
                   ELSE {"item", "data", "payload", "nested"}
                     \cup IF HasExtension(s) THEN {"outbound-schema"} ELSE {}
ExpectedPlan(s) == [kind |-> IF Branch(s) THEN "stream" ELSE "ordinary",
  payload |-> "P", default |-> "d1", assertion |-> "a1",
  keys |-> Source(s).keys, headers |-> <<"required","optional">>,
  statusField |-> s = "class2XX" \/ Two(s)]
ExpectedOutcome(s) == [
  root |-> [owner |-> "occurrence1", value |-> Project(s,RootRaw(s,1))],
  child |-> [owner |-> "declaration-child", value |-> Project(s,"sample")],
  cut |-> Source(s).cut]
Empty == [none |-> TRUE]

Init ==
        /\ scenario \in IF CaseFilter = "all" THEN Cases ELSE {CaseFilter}
        /\ order \in {<<1,2>>, <<2,1>>}
        /\ phase = "normalize" /\ loaded = Source(scenario)
        /\ retained = {} /\ closure = {} /\ visited = {}
        /\ diagnostics = {} /\ admitted = FALSE
        /\ plan = Empty /\ ordinary = Empty /\ derived = Empty
Normalize ==
  /\ phase = "normalize"
  /\ retained' = IF Mode = "current" /\ Branch(scenario) THEN {}
                  ELSE ExpectedEdges(scenario)
  /\ closure' = IF Mode = "current" /\ Branch(scenario) THEN {}
                 ELSE IF Mode = "literal-edge" THEN ExpectedEdges(scenario) \cup {"not-a-schema-edge"}
                 ELSE ExpectedEdges(scenario)
  /\ diagnostics' = IF Mode = "current" /\ Branch(scenario)
                     THEN {"media-item-schema"} ELSE {}
  /\ phase' = "declarations"
  /\ UNCHANGED <<scenario,order,loaded,visited,admitted,plan,ordinary,derived>>
Declarations ==
  /\ phase = "declarations"
  /\ diagnostics' = IF loaded.declarationContent /\ Mode # "permission-leak"
                     THEN diagnostics \cup {"content-keyword"} ELSE diagnostics
  /\ phase' = "occurrences"
  /\ UNCHANGED <<scenario,order,loaded,retained,closure,visited,admitted,plan,ordinary,derived>>
Visit ==
  /\ phase = "occurrences" /\ visited # Indices(scenario)
  /\ LET i == IF order[1] \in Indices(scenario) \ visited THEN order[1] ELSE order[2]
     IN visited' = visited \cup {i}
  /\ UNCHANGED <<scenario,order,phase,loaded,retained,closure,diagnostics,admitted,plan,ordinary,derived>>
Admit ==
  /\ phase = "occurrences" /\ visited = Indices(scenario)
  /\ LET bad == (loaded.item /\ loaded.dialect # "3.2")
           \/ (loaded.item /\ loaded.envelope # "closed")
           \/ loaded.mapped \/ loaded.projected
           \/ (~Equivalent(loaded) /\ Mode # "first-payload")
           \/ (loaded.extension /\ (~loaded.item \/ ~loaded.extensionMatches)
                                    /\ Mode # "stale-reserved")
         ok == diagnostics = {} /\ ~bad
     IN /\ admitted' = ok
        /\ diagnostics' = IF bad THEN diagnostics \cup {"unsupported"} ELSE diagnostics
  /\ phase' = "plan"
  /\ UNCHANGED <<scenario,order,loaded,retained,closure,visited,plan,ordinary,derived>>
Plan ==
  /\ phase = "plan"
  /\ plan' = IF admitted THEN ExpectedPlan(scenario) ELSE Empty
  /\ ordinary' = IF admitted THEN ExpectedOutcome(scenario) ELSE Empty
  /\ loaded' = IF Mode = "source-mutation" /\ admitted
                 THEN [loaded EXCEPT !.child.value = "overwritten"] ELSE loaded
  /\ phase' = "derive"
  /\ UNCHANGED <<scenario,order,retained,closure,visited,diagnostics,admitted,derived>>
Derive ==
  /\ phase = "derive"
  /\ derived' = IF ~admitted THEN Empty
       ELSE IF Mode = "wrong-owner"
         THEN [ordinary EXCEPT !.root.owner = "declaration-child"]
       ELSE IF Mode = "independent-example"
         THEN [ordinary EXCEPT !.child.value = "async-seed"]
       ELSE IF Mode = "suppression-fallback" /\ ordinary.root.value = Absent
         THEN [ordinary EXCEPT !.root.value = "sample"]
       ELSE IF Mode = "double-custom" /\ scenario = "custom"
         THEN [ordinary EXCEPT !.root.value = <<"projected", @>>]
       ELSE ordinary
  /\ phase' = "done"
  /\ UNCHANGED <<scenario,order,loaded,retained,closure,visited,diagnostics,admitted,plan,ordinary>>
Next == Normalize \/ Declarations \/ Visit \/ Admit \/ Plan \/ Derive
Spec == Init /\ [][Next]_vars

SourceUnchanged == loaded = Source(scenario)
RequiredSupport == phase = "done" /\ scenario \in Allowed => admitted
AdmissionSound == phase = "done" /\ admitted => scenario \in Allowed
ContractRetained == phase = "done" /\ admitted =>
                     plan = ExpectedPlan(scenario) /\ retained = ExpectedEdges(scenario)
TypedClosure == phase = "done" => closure = ExpectedEdges(scenario)
AnnotationAuthority == phase = "done" /\ admitted => derived = ExpectedOutcome(scenario)
CurrentRejects == Mode = "current" /\ phase = "done" /\ Branch(scenario) =>
                  ~admitted /\ "media-item-schema" \in diagnostics
CurrentRetention == phase # "normalize" /\ Branch(scenario) => "payload" \in retained
NoDone == ~(phase = "done")
NoAdmitted == ~(phase = "done" /\ admitted /\ plan.kind = "stream")
NoTwoReverse == ~(phase = "done" /\ admitted /\ scenario = "twoEqual" /\ order = <<2,1>>)
NoSuppressed == ~(phase = "done" /\ admitted /\ scenario = "suppressed" /\ derived.root.value = Absent)
NoCustom == ~(phase = "done" /\ admitted /\ scenario = "custom" /\ derived.root.value = <<"projected","sample">>)
NoRecursive == ~(phase = "done" /\ admitted /\ scenario = "recursive" /\ derived.cut = "P" /\ "outbound-schema" \in closure)
NoRejected == ~(phase = "done" /\ ~admitted /\ diagnostics # {})
=============================================================================
