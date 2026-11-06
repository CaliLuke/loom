------------------------ MODULE AdditionalProjection ------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Policy
VARIABLES captured, prepared, pending, projected, phase, key, retained
vars == <<captured, prepared, pending, projected, phase, key, retained>>
Nodes == {"record", "extras", "bytes", "recursive"}
Edges == {<<"record", "extras">>, <<"extras", "bytes">>,
          <<"extras", "recursive">>, <<"recursive", "extras">>}
Children(n) == {m \in Nodes : <<n, m>> \in Edges}
Consumers == {"schema", "example", "decoder"}
Init == /\ captured = {"record"} /\ pending = {"record"}
        /\ prepared = {} /\ projected = {} /\ phase = "capture"
        /\ key \in {"", "x"} /\ retained = FALSE
Capture == /\ phase = "capture" /\ pending # {}
           /\ \E n \in pending:
                /\ LET children == IF Policy = "drop" /\ n = "record"
                                      THEN {} ELSE Children(n)
                   IN /\ captured' = captured \cup children
                      /\ pending' = (pending \ {n}) \cup (children \ captured)
                /\ prepared' = prepared \cup {n}
           /\ UNCHANGED <<projected, phase, key, retained>>
Project == /\ phase = "capture" /\ pending = {}
           /\ phase' = "done"
           /\ retained' = ~(Policy = "name-sentinel" /\ key = "")
           /\ projected' = IF Policy = "schema-only"
                              THEN {"schema"} \X captured
                              ELSE Consumers \X captured
           /\ UNCHANGED <<captured, pending, prepared, key>>
Next == Capture \/ Project \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
Complete == phase = "done" => captured = Nodes
SharedRepresentation == phase = "done" => projected = Consumers \X Nodes
KeyPresence == phase = "done" => retained
\* Required keys belong to the captured occurrence, including aliases. A
\* shared declaration must never inherit a sibling occurrence's predicates.
OccurrenceKeys == [o \in {"base", "x", "y", "alias-x"} |->
    CASE o = "base" -> {}
      [] o = "y" -> {"y"}
      [] OTHER -> {"x"}]
CapturedKeys == IF Policy = "alias-drop"
                  THEN [o \in DOMAIN OccurrenceKeys |-> {}]
                ELSE IF Policy = "alias-mutate"
                  THEN [o \in DOMAIN OccurrenceKeys |-> {"x", "y"}]
                ELSE OccurrenceKeys
OccurrenceRequirements == phase = "done" => CapturedKeys = OccurrenceKeys
=============================================================================
