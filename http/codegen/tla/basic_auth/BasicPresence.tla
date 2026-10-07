------------------------- MODULE BasicPresence -------------------------
EXTENDS Naturals
CONSTANT Mode
VARIABLES required, fields, source, header, decoded, rejected, phase
vars == <<required, fields, source, header, decoded, rejected, phase>>
Components == {"user", "pass"}
Values == {"absent", "empty", "value"}
Present(v) == v # "absent"
Required == \E c \in Components: required[c]
Send == Required \/ (\E c \in Components: Present(fields[c]))
WireValues == [c \in Components |-> IF Present(fields[c]) THEN fields[c] ELSE "empty"]
Init == /\ required \in [Components -> BOOLEAN]
        /\ fields \in [Components -> Values]
        /\ \A c \in Components: required[c] => Present(fields[c])
        /\ source \in {"encoded", "missing", "invalid"}
        /\ header = FALSE
        /\ decoded = [c \in Components |-> "absent"]
        /\ rejected = FALSE
        /\ phase = 0
Encode == /\ phase = 0
          /\ header' = IF Mode = "legacy-encode"
                         THEN \A c \in Components: Present(fields[c])
                         ELSE Send
          /\ phase' = 1
          /\ UNCHANGED <<required, fields, source, decoded, rejected>>
ValidHeader == source = "encoded" /\ header
Decode == /\ phase = 1
          /\ rejected' = (Required /\ ~ValidHeader)
          /\ decoded' = IF ValidHeader THEN WireValues
                          ELSE IF Mode = "legacy-decode" /\ ~Required
                               THEN [c \in Components |-> "empty"]
                               ELSE [c \in Components |-> "absent"]
          /\ phase' = 2
          /\ UNCHANGED <<required, fields, source, header>>
Next == Encode \/ Decode \/ (phase = 2 /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
PreservesPresence == phase > 0 => header = Send
AbsentStaysAbsent == phase = 2 /\ ~ValidHeader /\ ~Required =>
                      \A c \in Components: decoded[c] = "absent"
RejectsRequired == phase = 2 => rejected = (Required /\ ~ValidHeader)
WireRoundTrip == phase = 2 /\ ValidHeader => decoded = WireValues
=======================================================================
