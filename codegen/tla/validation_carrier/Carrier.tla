-------------------------- MODULE Carrier --------------------------
EXTENDS Naturals
CONSTANT Mode
VARIABLES layout, extracted, named, phase, guard, childLayout
vars == <<layout, extracted, named, phase, guard, childLayout>>
Init == /\ layout \in BOOLEAN
        /\ extracted \in BOOLEAN
        /\ named \in BOOLEAN
        /\ (named \/ extracted \/ layout)
        /\ phase = "root"
        /\ guard = FALSE
        /\ childLayout = layout
\* Object fields are pointers, but an extracted object is a concrete value.
\* A forced field layout is independent from that occurrence's carrier.
Render == /\ phase = "root"
          /\ guard' = IF Mode = "legacy"
                       THEN IF named THEN ~extracted ELSE layout
                       ELSE ~extracted
          /\ childLayout' = IF Mode = "clear-layout" /\ extracted
                             THEN FALSE ELSE layout
          /\ phase' = "child"
          /\ UNCHANGED <<layout, extracted, named>>
Next == Render
Spec == Init /\ [][Next]_vars
TypedGuard == phase = "child" => (extracted => ~guard)
NativeGuard == phase = "child" => (~extracted => guard)
FieldLayout == phase = "child" => childLayout = layout
====================================================================
