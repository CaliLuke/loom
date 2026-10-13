---------------------- MODULE ProblemProperties ----------------------
EXTENDS Naturals, FiniteSets

CONSTANT Mode
Properties == {"timeout", "temporary", "fault"}
Guessed(status) == CASE status = "conflict" -> {}
                    [] status = "internal" -> {"fault"}
                    [] status = "gateway" -> {"timeout", "temporary"}
                    [] OTHER -> {}

VARIABLES authored, status, selected, phase, document, body, decoded
vars == <<authored, status, selected, phase, document, body, decoded>>

Init == /\ authored \in SUBSET Properties
        /\ status \in {"conflict", "internal", "gateway", "rpc"}
        /\ selected \in SUBSET Properties
        /\ phase = "source"
        /\ document = {}
        /\ body = {}
        /\ decoded = {}

Encode == /\ phase = "source"
          /\ document' = IF Mode = "current" THEN {} ELSE authored
          /\ phase' = "document"
          /\ UNCHANGED <<authored, status, selected, body, decoded>>

Project == /\ phase = "document"
           /\ body' = document \cap selected
           /\ phase' = "body"
           /\ UNCHANGED <<authored, status, selected, document, decoded>>

Decode == /\ phase = "body"
          /\ decoded' = IF Mode = "explicit" THEN body
                         ELSE Guessed(status)
          /\ phase' = "decoded"
          /\ UNCHANGED <<authored, status, selected, document, body>>

Next == Encode \/ Project \/ Decode
Spec == Init /\ [][Next]_vars
PreserveSelected == phase = "decoded" => decoded = (authored \cap selected)
NoInventedProperties == phase = "decoded" => decoded \subseteq authored
======================================================================
