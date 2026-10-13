----------------------- MODULE DomainAdmission -----------------------
EXTENDS Integers, TLC
CONSTANT Mode
VARIABLES kind, lower, upper, overlay, lowerExclusive, upperExclusive,
          inherited, admitted, phase
vars == <<kind, lower, upper, overlay, lowerExclusive, upperExclusive,
          inherited, admitted, phase>>
Init == /\ kind = "numeric" /\ lower = 0 /\ upper = 0 /\ overlay = 0
        /\ lowerExclusive = FALSE /\ upperExclusive = FALSE
        /\ inherited = FALSE /\ admitted = FALSE /\ phase = "choose"
Max(a,b) == IF a > b THEN a ELSE b
NumericValid == lower < upper \/ (lower = upper /\ ~lowerExclusive /\ ~upperExclusive)
LengthValid == lower >= 0 /\ overlay >= 0 /\ upper >= 0 /\ Max(lower,overlay) <= upper
NumericWitness == \E point \in -2..2:
 (IF lowerExclusive THEN point > 2*lower ELSE point >= 2*lower) /\
 (IF upperExclusive THEN point < 2*upper ELSE point <= 2*upper)
LengthWitness == lower >= 0 /\ overlay >= 0 /\ upper >= 0 /\
                 (\E n \in 0..1: n >= lower /\ n >= overlay /\ n <= upper)
Contract == IF kind = "numeric" THEN NumericWitness ELSE LengthWitness
LegacyNumeric == inherited \/ lower < upper \/
                 (lower = upper /\ (lowerExclusive = upperExclusive))
MergedLength == Max(lower,overlay) >= 0 /\ upper >= 0 /\ Max(lower,overlay) <= upper
Choose == /\ phase = "choose" /\ phase' = "admit"
 /\ kind' \in {"numeric","length"}
 /\ lower' \in -1..1 /\ upper' \in -1..1 /\ overlay' \in -1..1
 /\ lowerExclusive' \in BOOLEAN /\ upperExclusive' \in BOOLEAN
 /\ inherited' \in BOOLEAN /\ UNCHANGED admitted
Admit == /\ phase = "admit" /\ phase' = "done"
 /\ admitted' = (IF Mode = "local" THEN
                  IF kind = "numeric" THEN LegacyNumeric ELSE inherited \/ lower <= upper
                 ELSE IF Mode = "merged" THEN
                  IF kind = "numeric" THEN NumericValid ELSE MergedLength
                 ELSE IF kind = "numeric" THEN NumericValid ELSE LengthValid)
 /\ UNCHANGED <<kind,lower,upper,overlay,lowerExclusive,upperExclusive,inherited>>
Next == Choose \/ Admit \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
Admission == phase = "done" => admitted = Contract
=============================================================================
