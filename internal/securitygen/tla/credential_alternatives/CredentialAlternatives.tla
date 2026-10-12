--------------------- MODULE CredentialAlternatives ---------------------
EXTENDS Naturals, TLC
CONSTANT Legacy
VARIABLES phase, topology, framing, required, firstOK, secondOK, rejected, authorized, context
vars == <<phase, topology, framing, required, firstOK, secondOK, rejected, authorized, context>>
Alternative == topology \in {"or", "anonymous"}
ForeignAllowed == Alternative /\ ~required
Malformed == framing = "malformed"
AbsentRequired == required /\ framing = "absent"
ExpectedReject == Malformed \/ AbsentRequired \/ (framing = "foreign" /\ ~ForeignAllowed)
FirstValid == framing = "valid" /\ firstOK
ExpectedAuthorize == ~ExpectedReject /\
 IF topology = "and" THEN FirstValid /\ secondOK
 ELSE FirstValid \/ (topology = "or" /\ secondOK) \/ topology = "anonymous"
Init == /\ phase = "choose"
        /\ topology = "single" /\ framing = "absent" /\ required = FALSE
        /\ firstOK = FALSE /\ secondOK = FALSE
        /\ rejected = FALSE /\ authorized = FALSE /\ context = "original"
Choose == /\ phase = "choose"
          /\ topology' \in {"single", "and", "or", "shared", "anonymous"}
          /\ framing' \in {"absent", "foreign", "valid", "malformed"}
          /\ required' \in BOOLEAN /\ firstOK' \in BOOLEAN /\ secondOK' \in BOOLEAN
          /\ phase' = "decode"
          /\ UNCHANGED <<rejected, authorized, context>>
Decode == /\ phase = "decode"
          /\ rejected' = IF Legacy THEN Malformed \/ AbsentRequired \/ framing = "foreign" ELSE ExpectedReject
          /\ phase' = "authenticate"
          /\ UNCHANGED <<topology, framing, required, firstOK, secondOK, authorized, context>>
Authenticate == /\ phase = "authenticate"
                /\ authorized' = ~rejected /\
                    IF topology = "and" THEN FirstValid /\ secondOK
                    ELSE FirstValid \/ (topology = "or" /\ secondOK) \/ topology = "anonymous"
                /\ context' = IF authorized' THEN
                     IF topology = "and" THEN "first-then-second"
                     ELSE IF FirstValid THEN "first"
                     ELSE IF topology = "or" THEN "original-then-second"
                     ELSE "original"
                    ELSE "original"
                /\ phase' = "done"
                /\ UNCHANGED <<topology, framing, required, firstOK, secondOK, rejected>>
Next == Choose \/ Decode \/ Authenticate
Spec == Init /\ [][Next]_vars
OptionalAlternativeUsable == phase = "done" => (authorized = ExpectedAuthorize)
StrictRejection == phase = "done" /\ ExpectedReject => ~authorized
ContextIsolation == phase = "done" /\ authorized /\ ~FirstValid /\ topology = "or" => context = "original-then-second"
AndContext == phase = "done" /\ authorized /\ topology = "and" => context = "first-then-second"
=============================================================================
