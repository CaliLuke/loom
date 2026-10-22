--------------------------- MODULE CompactNumber ---------------------------
EXTENDS Integers, Naturals
CONSTANT WrapOverflow
VARIABLES original, signum, coefficient, e2, e5, phase, magnitude, left2, left5
vars == <<original, signum, coefficient, e2, e5, phase, magnitude, left2, left5>>
Maximum == 255
Init == /\ coefficient \in 0..31
        /\ e2 \in 0..4
        /\ e5 \in 0..4
        /\ signum \in {-1,1}
        /\ original = signum * coefficient * 2^e2 * 5^e5
        /\ phase = "normalize"
        /\ magnitude = 0 /\ left2 = 0 /\ left5 = 0
Factor(base) ==
 /\ phase = "normalize" /\ coefficient > 0 /\ coefficient % base = 0
 /\ coefficient' = coefficient \div base
 /\ e2' = IF base = 2 THEN e2 + 1 ELSE e2
 /\ e5' = IF base = 5 THEN e5 + 1 ELSE e5
 /\ UNCHANGED <<original, signum, phase, magnitude, left2, left5>>
Canonical == coefficient = 0 \/ (coefficient % 2 # 0 /\ coefficient % 5 # 0)
Begin ==
 /\ phase = "normalize" /\ Canonical
 /\ phase' = "reconstruct"
 /\ magnitude' = coefficient /\ left2' = e2 /\ left5' = e5
 /\ UNCHANGED <<original, signum, coefficient, e2, e5>>
Multiply(base) ==
 /\ phase = "reconstruct"
 /\ IF base = 2 THEN left2 > 0 ELSE left2 = 0 /\ left5 > 0
 /\ LET product == magnitude * base
    IN /\ magnitude' = IF WrapOverflow THEN product % (Maximum + 1) ELSE product
       /\ phase' = IF product > Maximum /\ ~WrapOverflow THEN "fallback" ELSE phase
 /\ left2' = IF base = 2 THEN left2 - 1 ELSE left2
 /\ left5' = IF base = 5 THEN left5 - 1 ELSE left5
 /\ UNCHANGED <<original, signum, coefficient, e2, e5>>
Finish ==
 /\ phase = "reconstruct" /\ left2 = 0 /\ left5 = 0
 /\ phase' = "native"
 /\ UNCHANGED <<original, signum, coefficient, e2, e5, magnitude, left2, left5>>
Next == Factor(2) \/ Factor(5) \/ Begin \/ Multiply(2) \/ Multiply(5) \/ Finish
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
NormalizedMeaning == original = signum * coefficient * 2^e2 * 5^e5
ExactNative == phase = "native" => signum * magnitude = original /\ magnitude <= Maximum
NecessaryFallback == phase = "fallback" => original > Maximum \/ original < -Maximum
Completes == <> (phase \in {"native","fallback"})
=============================================================================
