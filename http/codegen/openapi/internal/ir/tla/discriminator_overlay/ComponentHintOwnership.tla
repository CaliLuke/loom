---------------------- MODULE ComponentHintOwnership ----------------------
EXTENDS Naturals
CONSTANT Policy
VARIABLES annotated, wrapper, kind, localHint, sourceHint, componentHint,
          hintOwner, annotationOwner, phase
vars == <<annotated, wrapper, kind, localHint, sourceHint, componentHint,
          hintOwner, annotationOwner, phase>>
Init == /\ annotated \in BOOLEAN /\ wrapper \in {"ref", "allOf"}
        /\ kind \in {"union", "parent"} /\ localHint \in {"base", "local"}
        /\ sourceHint = "base" /\ componentHint = "base"
        /\ hintOwner = "unset" /\ annotationOwner = "unset" /\ phase = "ready"
Next == /\ phase = "ready" /\ phase' = "done"
        /\ hintOwner' = IF annotated /\ Policy = "overlay" THEN wrapper ELSE "component"
        /\ componentHint' = IF annotated /\ Policy \in {"overlay", "drop"} THEN "base" ELSE localHint
        /\ sourceHint' = IF Policy = "mutate" THEN localHint ELSE sourceHint
        /\ annotationOwner' = IF annotated THEN wrapper ELSE "none"
        /\ UNCHANGED <<annotated, wrapper, kind, localHint>>
CompleteOwner == phase = "done" => hintOwner = "component"
HintRetention == phase = "done" => componentHint = localHint
SourceIsolation == sourceHint = "base"
AnnotationIsolation == phase = "done" /\ annotated => annotationOwner = wrapper
Spec == Init /\ [][Next]_vars
=============================================================================
