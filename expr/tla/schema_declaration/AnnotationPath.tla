------------------------- MODULE AnnotationPath -------------------------
EXTENDS Naturals, Sequences, TLC
CONSTANT Mode
VARIABLES baseline, insertion, addedLayers, sampled, cursor, remaining,
          emitted, phase
vars == <<baseline, insertion, addedLayers, sampled, cursor, remaining,
          emitted, phase>>

\* One finite observed schema path. Zero records absence of an example;
\* nonzero labels stand for complete annotation payloads, not numeric samples.
Samples == {0, 1, 2}
Paths == UNION {[1..length -> Samples] : length \in 1..3}
OriginalNode(index) == [owner |-> index, annotation |-> baseline[index]]
OriginalPath == [index \in 1..Len(baseline) |-> OriginalNode(index)]
Observed(path) == SelectSeq(path, LAMBDA node: node.annotation # 0)

Init ==
    /\ baseline \in Paths
    /\ insertion \in 1..Len(baseline)
    /\ addedLayers \in 0..2
    /\ sampled \in {1, 2}
    /\ cursor = 1
    /\ remaining = addedLayers
    /\ emitted = <<>>
    /\ phase = "traverse"

\* New schema layers can be needed for representation references even when
\* their DSL aliases existed before. Their appearance is not sample authority.
InsertLayer ==
    /\ phase = "traverse"
    /\ cursor = insertion /\ remaining > 0
    /\ emitted' = Append(emitted,
          [owner |-> 0,
           annotation |-> IF Mode = "reconstruct" THEN sampled ELSE 0])
    /\ remaining' = remaining - 1
    /\ UNCHANGED <<baseline, insertion, addedLayers, sampled, cursor, phase>>

VisitOriginal ==
    /\ phase = "traverse" /\ cursor <= Len(baseline)
    /\ (cursor # insertion \/ remaining = 0)
    /\ emitted' = Append(emitted,
          [owner |-> cursor,
           annotation |-> IF Mode = "drop-authored" /\ cursor < Len(baseline)
                          THEN 0
                          ELSE IF Mode = "resample-absent"
                                  /\ cursor < Len(baseline) /\ baseline[cursor] = 0
                               THEN sampled ELSE baseline[cursor]])
    /\ cursor' = cursor + 1
    /\ UNCHANGED <<baseline, insertion, addedLayers, sampled, remaining, phase>>
Finish ==
    /\ phase = "traverse" /\ cursor > Len(baseline)
    /\ phase' = "done"
    /\ UNCHANGED <<baseline, insertion, addedLayers, sampled, cursor,
                    remaining, emitted>>
Next == InsertLayer \/ VisitOriginal \/ Finish
Spec == Init /\ [][Next]_vars

\* The earlier scalar-owner check passes even for both defective policies.
TerminalOwnerPreserved ==
    phase = "done" => emitted[Len(emitted)] = OriginalNode(Len(baseline))
\* Original positions and payloads remain in order, including the obligation
\* that newly materialized layers cannot create previously absent annotations.
AnnotationPathPreserved ==
    phase = "done" => Observed(emitted) = Observed(OriginalPath)
OriginalAbsencePreserved ==
    phase = "done" =>
      \A index \in 1..Len(emitted):
        emitted[index].owner # 0 =>
          (baseline[emitted[index].owner] = 0 => emitted[index].annotation = 0)
NewLayersUnannotated ==
    phase = "done" =>
      \A index \in 1..Len(emitted):
        emitted[index].owner = 0 => emitted[index].annotation = 0
=============================================================================
