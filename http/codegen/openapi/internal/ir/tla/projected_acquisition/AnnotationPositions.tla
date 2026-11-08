-------------------- MODULE AnnotationPositions --------------------
EXTENDS Naturals, Sequences, FiniteSets
CONSTANT Policy
VARIABLES visits, memo, outputs, acquisitions
vars == <<visits, memo, outputs, acquisitions>>
Positions == 1..3
Values == [p \in Positions |-> IF p = 1 THEN "declaration" ELSE "local"]
Keys == 0..3
Key(p) == IF Policy = "context-shape" THEN 0 ELSE p
Init == /\ visits = [p \in Positions |-> 0]
        /\ memo = [k \in Keys |-> "absent"]
        /\ outputs = <<>>
        /\ acquisitions = [k \in Keys |-> 0]
Read == \E p \in Positions:
        /\ visits[p] < 2
        /\ LET k == Key(p)
               value == IF memo[k] = "absent" THEN Values[p] ELSE memo[k]
           IN /\ memo' = [memo EXCEPT ![k] = value]
              /\ acquisitions' = [acquisitions EXCEPT ![k] = @ + (IF memo[k] = "absent" THEN 1 ELSE 0)]
              /\ outputs' = Append(outputs, [position |-> p, value |-> value])
        /\ visits' = [visits EXCEPT ![p] = @ + 1]
Spec == Init /\ [][Read]_vars
TypeOK == /\ visits \in [Positions -> 0..2]
          /\ memo \in [Keys -> {"absent", "declaration", "local"}]
          /\ acquisitions \in [Keys -> 0..1]
          /\ outputs \in Seq([position: Positions, value: {"declaration", "local"}])
OwnValue == \A i \in 1..Len(outputs): outputs[i].value = Values[outputs[i].position]
AtMostOnce == \A k \in Keys: acquisitions[k] <= 1
RepeatedBinding == \A i,j \in 1..Len(outputs): outputs[i].position = outputs[j].position => outputs[i].value = outputs[j].value
Complete == \A p \in Positions: visits[p] = 2
SemanticDedup == Complete => Cardinality({outputs[i].value : i \in 1..Len(outputs)}) = 2
NoComplete == ~Complete
====================================================================
