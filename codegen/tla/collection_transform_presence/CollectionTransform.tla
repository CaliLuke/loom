-------------------------- MODULE CollectionTransform --------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
Cases == [sourceJSON: BOOLEAN, targetJSON: BOOLEAN, nested: BOOLEAN,
          named: BOOLEAN, supplied: BOOLEAN, size: 0..2, mapping: BOOLEAN, missing: BOOLEAN]
VARIABLES current, done, targetSupplied, targetSize, outerWrapped, innerWrapped
vars == <<current, done, targetSupplied, targetSize, outerWrapped, innerWrapped>>
Init == /\ current \in {c \in Cases: c.supplied \/ c.size = 0}
        /\ done = FALSE /\ targetSupplied = FALSE /\ targetSize = 0
        /\ outerWrapped = FALSE /\ innerWrapped = FALSE
Convert ==
 /\ ~done /\ done' = TRUE /\ UNCHANGED current
 /\ targetSupplied' = current.supplied
 /\ targetSize' = IF Mode = "drop-slot" /\ current.mapping /\ current.sourceJSON /\ current.missing /\ current.size > 0 THEN current.size - 1 ELSE current.size
 /\ outerWrapped' = CASE Mode = "source-only" -> FALSE
                       [] Mode = "alias-cast" /\ current.named -> current.sourceJSON
                       [] OTHER -> current.targetJSON
 /\ innerWrapped' = CASE Mode \in {"source-only", "outer-only"} -> FALSE
                       [] Mode = "alias-cast" /\ current.named -> current.sourceJSON
                       [] OTHER -> current.targetJSON
Next == Convert \/ (done /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
ContainerPreserved == done => targetSupplied = current.supplied /\ targetSize = current.size
TargetCarrier == done /\ current.size > 0 => outerWrapped = current.targetJSON
NestedCarrier == done /\ current.nested /\ current.size > 0 => innerWrapped = current.targetJSON
Terminates == <>done
================================================================================
