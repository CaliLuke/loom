------------------------ MODULE SourceApplication ------------------------
EXTENDS Naturals
CONSTANT Policy
VARIABLES outer, item, media, flags, location, application, selected, phase
vars == <<outer, item, media, flags, location, application, selected, phase>>
Types == {"absent", "scalar", "object", "array"}
Default(t) == CASE t = "scalar" -> "text" [] t \in {"object", "array"} -> "json" [] OTHER -> "octets"
ExpectedApplication == IF flags \/ outer # "array" THEN "value" ELSE "items"
ExpectedMedia == IF flags THEN "style"
                 ELSE IF media # "absent" THEN media
                 ELSE Default(IF outer = "array" THEN item ELSE outer)
Init == /\ outer \in Types /\ item \in Types
        /\ media \in {"absent", "text", "json", "octets"}
        /\ flags \in BOOLEAN /\ location \in {"form", "multipart"}
        /\ application = "unset" /\ selected = "unset" /\ phase = "capture"
Resolve == /\ phase = "capture"
           /\ application' = IF Policy = "whole-array" THEN "value" ELSE ExpectedApplication
           /\ selected' = IF flags THEN
                             IF Policy = "media-conflict" /\ location = "multipart" /\ media = "json"
                             THEN "reject" ELSE "style"
                           ELSE IF media # "absent" THEN media
                           ELSE Default(IF outer = "array" /\ Policy # "whole-array" THEN item ELSE outer)
           /\ phase' = "done"
           /\ UNCHANGED <<outer, item, media, flags, location>>
Next == Resolve \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
ApplicationAgreement == phase = "done" => application = ExpectedApplication
MediaAgreement == phase = "done" => selected = ExpectedMedia
=============================================================================
