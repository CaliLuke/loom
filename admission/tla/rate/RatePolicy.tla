------------------------- MODULE RatePolicy -------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
VARIABLES rate, burst, tokens, spent, refilled, steps, target, phase
vars == <<rate, burst, tokens, spent, refilled, steps, target, phase>>
Policies == {<<1, 2>>, <<2, 1>>, <<1, 3>>}
Min(a, b) == IF a < b THEN a ELSE b
Init == /\ rate = 1
        /\ burst = 2
        /\ tokens = 2
        /\ spent = 0
        /\ refilled = 0
        /\ steps = 0
        /\ target = <<1, 2>>
        /\ phase = "ready"
Consume == /\ tokens > 0
           /\ tokens' = tokens - 1
           /\ spent' = spent + 1
           /\ steps' = steps + 1
           /\ UNCHANGED <<rate, burst, refilled, target, phase>>
Refill == /\ tokens' = Min(burst, tokens + rate)
          /\ refilled' = refilled + (tokens' - tokens)
          /\ steps' = steps + 1
          /\ UNCHANGED <<rate, burst, spent, target, phase>>
Update(p) == /\ phase = "ready"
             /\ rate' = p[1]
             /\ burst' = IF Mode = "split" THEN burst ELSE p[2]
             /\ tokens' = CASE Mode = "reset" -> p[2]
                            [] Mode = "split" -> tokens
                            [] OTHER -> Min(tokens, p[2])
             /\ target' = p
             /\ phase' = IF Mode = "split" THEN "updating" ELSE "ready"
             /\ steps' = steps + 1
             /\ UNCHANGED <<spent, refilled>>
FinishUpdate == /\ phase = "updating"
                /\ burst' = target[2]
                /\ tokens' = Min(tokens, target[2])
                /\ phase' = "ready"
                /\ steps' = steps + 1
                /\ UNCHANGED <<rate, spent, refilled, target>>
Next == /\ steps < 5
        /\ Consume \/ Refill \/ (\E p \in Policies : Update(p)) \/ FinishUpdate
Spec == Init /\ [][Next]_vars
Conservation == spent + tokens <= 2 + refilled
CoherentPolicy == <<rate, burst>> \in Policies
Bounded == tokens <= burst
=====================================================================
