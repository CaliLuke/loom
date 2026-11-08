------------------------- MODULE ProjectionPublication -------------------------
EXTENDS Naturals, FiniteSets, TLC
CONSTANT ReserveReferenceFirst
VARIABLES phase, wrapperRef, componentReserved, componentComplete, published
vars == <<phase, wrapperRef, componentReserved, componentComplete, published>>

(* One memo key denotes (baseline schema node, immutable plan position,
   complete projection context). Parent publishes Child while Child's
   composition refers back to Parent. projectedSchemas owns a cached reference
   wrapper; projectedComponents separately owns the reserved component name.
   A recursive hit on the wrapper cache returns its current Ref immediately.
   The baseline Child encodes bytes as binary, the completed projected Child
   encodes them as base64. Initial copied Ref is therefore not interchangeable
   with the projected Ref, even if the new component name is already reserved. *)
Init == phase = 0 /\ wrapperRef = "absent" /\ componentReserved = FALSE /\ componentComplete = FALSE /\ published = {}
Next ==
  CASE phase = 0 -> /\ wrapperRef' = "baseline"
                      /\ phase' = 1
                      /\ UNCHANGED <<componentReserved, componentComplete, published>>
    [] phase = 1 -> /\ componentReserved' = TRUE
                      /\ wrapperRef' = (IF ReserveReferenceFirst THEN "projected" ELSE wrapperRef)
                      /\ phase' = 2
                      /\ UNCHANGED <<componentComplete, published>>
    [] phase = 2 -> /\ published' = published \cup {wrapperRef}
                      /\ phase' = 3
                      /\ UNCHANGED <<wrapperRef, componentReserved, componentComplete>>
    [] phase = 3 -> /\ componentComplete' = TRUE
                      /\ phase' = 4
                      /\ UNCHANGED <<wrapperRef, componentReserved, published>>
    [] phase = 4 -> /\ wrapperRef' = "projected"
                      /\ phase' = 5
                      /\ UNCHANGED <<componentReserved, componentComplete, published>>
    [] OTHER -> UNCHANGED vars
Spec == Init /\ [][Next]_vars
TypeOK == phase \in 0..5 /\ wrapperRef \in {"absent", "baseline", "projected"} /\ componentReserved \in BOOLEAN /\ componentComplete \in BOOLEAN /\ published \subseteq {"baseline", "projected"}
StableRecursiveIdentity == phase >= 2 => wrapperRef = "projected" /\ componentReserved
NoBaselinePublication == published \subseteq {"projected"}
CompletePublication == phase = 5 => published = {"projected"} /\ componentComplete
=============================================================================
