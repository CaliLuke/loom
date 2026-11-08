---------------------------- MODULE BodyNameHint ----------------------------
EXTENDS Naturals, Sequences, FiniteSets, TLC
CONSTANT Mode
VARIABLES scenario, phase, desired, assigned, sampling
vars == <<scenario, phase, desired, assigned, sampling>>

(* Two independently acquired components. The first is registered before the
   body, matching BuildBodyTypes. Complete graphs include retained annotations
   and publication dependencies; equality is an input, not inferred from shape.
   Samples are not regenerated or substituted during name selection. *)
Cases == {"equal", "annotations", "publication", "partial", "foreign", "sameLogical"}
Semantic(c) == IF c = "partial" THEN "partial" ELSE "shape"
Hint(c) == IF c = "foreign" THEN "Other" ELSE "RT"
Logical(c) == IF c = "sameLogical" THEN "declaration" ELSE "body"
Complete(c) == IF c \in {"annotations", "publication", "partial"} THEN c ELSE "graph"
Compatible(c) == Hint(c) = "RT"
Eligible(c) == Semantic(c) = "shape" /\ Compatible(c)
ChooseName(c) ==
 IF Semantic(c) = "shape" /\
    (Logical(c) = "declaration" \/ (Mode # "current" /\ (Compatible(c) \/ Mode = "ignoreHint")))
 THEN "RT" ELSE IF Hint(c) = "Other" THEN "Other_semantic" ELSE "RT_semantic"
AcquiredSampleName(c) == IF Semantic(c) = "shape" /\ Logical(c) = "declaration" THEN "RT" ELSE IF Hint(c) = "Other" THEN "Other_semantic" ELSE "RT_semantic"
Init == scenario \in Cases /\ phase = 0 /\ desired = "" /\ assigned = "" /\ sampling = ""
Next ==
 IF phase = 0
 THEN /\ desired' = ChooseName(scenario) /\ phase' = 1
      /\ sampling' = IF Mode = "coupled" THEN ChooseName(scenario) ELSE AcquiredSampleName(scenario)
      /\ UNCHANGED <<scenario, assigned>>
 ELSE IF phase = 1
 THEN /\ assigned' = IF desired = "RT" /\ Complete(scenario) # "graph" /\ Mode # "shapeOnly"
                      THEN "RT_complete" ELSE desired
      /\ phase' = 2 /\ UNCHANGED <<scenario, desired, sampling>>
 ELSE UNCHANGED vars
Spec == Init /\ [][Next]_vars
TypeOK == scenario \in Cases /\ phase \in 0..2
CompatibleDesired == phase >= 1 /\ Eligible(scenario) => desired = "RT"
ForeignNamesRemainDistinct == phase >= 1 /\ ~Compatible(scenario) => desired # "RT"
NoFalseMerge == phase = 2 /\ assigned = "RT" => Complete(scenario) = "graph"
EqualCompleteGraphMerges == phase = 2 /\ Eligible(scenario) /\ Complete(scenario) = "graph" => assigned = "RT"
PartialShapeKeepsOwnName == phase >= 1 /\ Semantic(scenario) # "shape" => desired # "RT"
AcquiredSamplePreserved == phase >= 1 => sampling = AcquiredSampleName(scenario)
=============================================================================
