-------------------- MODULE MultipartSpelling --------------------
EXTENDS Naturals, Sequences
CONSTANT Literal
VARIABLE input, kind, outcome, done
vars == <<input, kind, outcome, done>>
Tokens == {"word", "a+b", "a%2Cb", "a,b"}
URI(s) == CASE s = "a+b" -> "a b"
            [] s = "a%2Cb" -> "a,b"
            [] OTHER -> s
Init == /\ input \in Tokens /\ kind \in {"scalar", "delimited"}
        /\ outcome = "unset" /\ done = FALSE
Next == /\ ~done
        /\ outcome' = IF kind = "delimited" /\ input = "a,b"
                         THEN "unspellable"
                         ELSE IF Literal THEN input ELSE URI(input)
        /\ done' = TRUE /\ UNCHANGED <<input, kind>>
Spec == Init /\ [][Next]_vars
LiteralIdentity == done => outcome = IF kind = "delimited" /\ input = "a,b"
                                      THEN "unspellable" ELSE input
=============================================================================
