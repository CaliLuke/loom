---------------------- MODULE ResponseFieldStorage ----------------------
EXTENDS TLC
CONSTANT Mode
VARIABLE specimen, addressed, complete
vars == <<specimen, addressed, complete>>

\* These finite declaration facts are inputs, not a reimplementation or proof
\* of ServiceFieldLayout. Go seam and generated compile tests check correspondence.
Case(name, primitive, semantic, context, layout) ==
    [name |-> name, primitive |-> primitive, semantic |-> semantic,
     context |-> context, layout |-> layout]
Cases == {
    Case("required view scalar", TRUE, FALSE, TRUE, TRUE),
    Case("optional view scalar", TRUE, TRUE, TRUE, TRUE),
    Case("defaulted view scalar", TRUE, FALSE, TRUE, TRUE),
    Case("required view named scalar", TRUE, FALSE, TRUE, TRUE),
    Case("required view nullable scalar", TRUE, FALSE, TRUE, FALSE),
    Case("optional view nullable scalar", TRUE, TRUE, TRUE, FALSE),
    Case("view array", FALSE, FALSE, TRUE, FALSE),
    Case("view bytes", TRUE, FALSE, FALSE, FALSE),
    Case("required service scalar", TRUE, FALSE, FALSE, FALSE),
    Case("optional service scalar", TRUE, TRUE, TRUE, TRUE),
    Case("optional service nullable scalar", TRUE, TRUE, TRUE, FALSE)
}

Init == /\ specimen \in Cases
        /\ addressed = FALSE
        /\ complete = FALSE

Construct == /\ ~complete
             /\ addressed' = CASE Mode = "service" -> specimen.semantic
                                   [] Mode = "context" -> specimen.context
                                   [] OTHER -> specimen.primitive /\ specimen.layout
             /\ complete' = TRUE
             /\ UNCHANGED specimen

Next == Construct
Spec == Init /\ [][Next]_vars
TypeOK == /\ specimen \in Cases
          /\ addressed \in BOOLEAN
          /\ complete \in BOOLEAN

\* Scalar transforms yield a value; exactly the outer pointer chosen by the
\* declaration must be added. Arrays/bytes/wrappers retain their own storage.
AssignmentMatchesDeclaration == complete =>
    addressed = (specimen.primitive /\ specimen.layout)
=============================================================================
