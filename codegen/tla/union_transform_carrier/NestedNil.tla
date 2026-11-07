---------------------------- MODULE NestedNil ----------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
Inputs == {"absent", "nil", "empty", "value"}
Cases == {c \in [sourcePointer: BOOLEAN, targetPointer: BOOLEAN, input: Inputs]:
          c.sourcePointer \/ c.input # "nil"}
VARIABLES current, done, crashed, selected, nested
vars == <<current, done, crashed, selected, nested>>
Init == /\ current \in Cases /\ done = FALSE /\ crashed = FALSE
        /\ selected = FALSE /\ nested = "absent"
Convert ==
  /\ ~done /\ done' = TRUE /\ UNCHANGED current
  /\ crashed' = (Mode = "old" /\ current.input = "nil")
  /\ selected' = (current.input # "absent" /\ ~(Mode = "drop" /\ current.input = "nil"))
  /\ nested' = IF current.input = "nil"
                THEN IF current.targetPointer THEN "nil" ELSE "empty"
                ELSE current.input
Next == Convert \/ (done /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
NoPanic == ~crashed
OuterSelectionPreserved == done => (selected = (current.input # "absent"))
NilPreserved == done /\ current.input = "nil" =>
                 nested = IF current.targetPointer THEN "nil" ELSE "empty"
ConcretePreserved == done /\ current.input = "value" => nested = "value"
Terminates == <>done
=============================================================================
