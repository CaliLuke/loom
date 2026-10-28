---------------------- MODULE ReusableReferences ----------------------
EXTENDS TLC
CONSTANT Mode
VARIABLES kind, name, localDescription, localSummary, edges
Kinds == {"requestBodies", "responses", "headers"}
Names == {"A", "B"}
Absent == "absent"
Texts == {Absent, "", "local"}
RECURSIVE Resolve(_, _)
Resolve(current, seen) == IF current \in seen THEN "cycle"
                         ELSE IF edges[current] = "terminal" THEN current
                         ELSE Resolve(edges[current], seen \cup {current})
Resolved == Resolve(name, {})
Base == [description |-> "base", summary |-> "base-summary", content |-> "wire"]
Init == /\ kind \in Kinds /\ name \in Names
        /\ localDescription \in Texts /\ localSummary \in Texts
        /\ edges \in [Names -> Names \cup {"terminal"}]
Description == IF localDescription = Absent THEN Base.description ELSE localDescription
Summary == IF localSummary = Absent THEN Base.summary ELSE localSummary
Admitted == Mode # "reject" /\ (Mode = "cycle" \/ Resolved # "cycle")
Reference == IF Mode = "inline" THEN <<kind, "lost">> ELSE <<kind, name>>
PublishedBase == IF Mode = "mutate" THEN [Base EXCEPT !.description = Description] ELSE Base
Occurrence == [description |-> Description, summary |-> Summary, content |-> Base.content]
SupportedInputAdmitted == Resolved # "cycle" => Admitted
CycleRejected == Resolved = "cycle" => ~Admitted
IdentityPreserved == Reference = <<kind, name>>
BaseUnchanged == PublishedBase = Base
OverlayApplied == Occurrence.description = Description /\ Occurrence.summary = Summary
ContentPreserved == Occurrence.content = Base.content
Next == UNCHANGED <<kind, name, localDescription, localSummary, edges>>
Spec == Init /\ [][Next]_<<kind, name, localDescription, localSummary, edges>>
======================================================================
