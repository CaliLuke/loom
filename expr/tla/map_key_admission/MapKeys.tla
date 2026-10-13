------------------------------ MODULE MapKeys ------------------------------
EXTENDS Naturals, Sequences
CONSTANT Strategy
VARIABLES shape, order, position, seen, keyAccepted, valueAccepted
vars == <<shape, order, position, seen, keyAccepted, valueAccepted>>

\* One declaration reused as ordinary data and as a map key, in either order.
\* Opaque means a valid externally owned Go representation, not proof that an
\* arbitrary authored codec is comparable. Object storage is a pointer.
Shapes == {"scalar", "slice", "object-pointer", "opaque"}
Init == /\ shape \in Shapes
        /\ order \in {<<"value", "key">>, <<"key", "value">>}
        /\ position = 1 /\ seen = FALSE
        /\ keyAccepted = FALSE /\ valueAccepted = FALSE
Allowed(role) == CASE Strategy = "current" -> TRUE
 [] Strategy = "primitive-only" -> role = "value" \/ shape = "scalar"
 [] Strategy = "global-cache" /\ seen -> TRUE
 [] OTHER -> role = "value" \/ shape # "slice"
Visit == /\ position <= 2
         /\ LET role == order[position] IN
              /\ keyAccepted' = IF role = "key" THEN Allowed(role) ELSE keyAccepted
              /\ valueAccepted' = IF role = "value" THEN Allowed(role) ELSE valueAccepted
         /\ seen' = TRUE /\ position' = position + 1
         /\ UNCHANGED <<shape, order>>
Done == /\ position = 3 /\ UNCHANGED vars
Next == Visit \/ Done
Spec == Init /\ [][Next]_vars
KeySafety == position = 3 /\ keyAccepted => shape # "slice"
PreserveContracts == position = 3 =>
 valueAccepted /\ (shape # "slice" => keyAccepted)
=============================================================================
