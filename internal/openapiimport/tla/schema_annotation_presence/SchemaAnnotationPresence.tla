--------------------- MODULE SchemaAnnotationPresence ---------------------
EXTENDS Naturals, TLC
CONSTANT Policy
VARIABLES phase, contributor, occurrence, contributorMeta, occurrenceMeta
vars == <<phase, contributor, occurrence, contributorMeta, occurrenceMeta>>

\* True programmatic values remain expressible without a parsed-presence bit.
AnnotationStates == [present: BOOLEAN, value: BOOLEAN]
Absent == "absent"
Text(state) == IF state.value THEN "true" ELSE "false"
Expected(state) == IF state.present \/ state.value THEN Text(state) ELSE Absent
OpenBlock(state) == IF Policy = "presence" THEN state.present \/ state.value
                    ELSE state.value
Emit(state) == IF Policy = "truthy" THEN IF state.value THEN "true" ELSE Absent
               ELSE Expected(state)
Render(state) == IF OpenBlock(state) THEN Emit(state) ELSE Absent

Init == /\ phase = "render"
        /\ contributor \in AnnotationStates
        /\ occurrence \in AnnotationStates
        /\ contributorMeta = "pending"
        /\ occurrenceMeta = "pending"
Publish == /\ phase = "render"
           /\ contributorMeta' = Render(contributor)
           /\ occurrenceMeta' = Render(occurrence)
           /\ phase' = "done"
           /\ UNCHANGED <<contributor, occurrence>>
Next == Publish \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars

ContributorPreserved == phase = "done" => contributorMeta = Expected(contributor)
OccurrencePreserved == phase = "done" => occurrenceMeta = Expected(occurrence)
Effective(local, base) == IF local = Absent THEN base ELSE local
EffectivePreserved == phase = "done" =>
  Effective(occurrenceMeta, contributorMeta) =
  Effective(Expected(occurrence), Expected(contributor))
=============================================================================
