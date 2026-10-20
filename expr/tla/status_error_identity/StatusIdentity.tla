-------------------------- MODULE StatusIdentity --------------------------
EXTENDS FiniteSets
CONSTANT Mode
VARIABLES statusSource, owners, wireSource, present, validName, admitted, decoded, phase
vars == <<statusSource, owners, wireSource, present, validName, admitted, decoded, phase>>

Init == /\ statusSource \in BOOLEAN
        /\ owners \in SUBSET {"errorA", "errorB", "success"}
        /\ "errorA" \in owners
        /\ wireSource \in {"none", "body", "header", "cookie"}
        /\ present \in BOOLEAN
        /\ validName \in BOOLEAN
        /\ admitted = FALSE
        /\ decoded = FALSE
        /\ phase = "admit"

Admit == /\ phase = "admit"
         /\ admitted' = IF statusSource
                        THEN owners = {"errorA"} /\ wireSource = "none" /\ validName
                        ELSE wireSource # "none"
         /\ phase' = "decode"
         /\ UNCHANGED <<statusSource, owners, wireSource, present, validName, decoded>>

Decode == /\ phase = "decode"
          /\ decoded' = admitted /\ (statusSource \/ present \/
                        (Mode = "fallback" /\ wireSource = "header" /\ owners = {"errorA"}))
          /\ phase' = "done"
          /\ UNCHANGED <<statusSource, owners, wireSource, present, validName, admitted>>

Next == Admit \/ Decode
Spec == Init /\ [][Next]_vars
RequiredWireRemainsRequired == phase = "done" /\ decoded /\ ~statusSource => present
StatusHasOneSource == admitted /\ statusSource =>
                     owners = {"errorA"} /\ wireSource = "none" /\ validName
ValidStatusDecodesWithoutHeader == phase = "done" /\ statusSource /\ admitted => decoded
=============================================================================
