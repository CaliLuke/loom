---------------------- MODULE ResponseBodyOwnership ----------------------
EXTENDS Naturals
CONSTANT Policy
VARIABLES parent, selected, replacement, metadata, tagged, phase,
          accepted, tagFieldRead, metadataFieldRead
vars == <<parent, selected, replacement, metadata, tagged, phase,
          accepted, tagFieldRead, metadataFieldRead>>
Init == /\ parent \in {"absent", "null", "value"}
        /\ selected \in BOOLEAN
        /\ replacement \in BOOLEAN
        /\ metadata \in BOOLEAN
        /\ tagged \in BOOLEAN
        /\ phase = "ready"
        /\ accepted = FALSE
        /\ tagFieldRead = FALSE
        /\ metadataFieldRead = FALSE
NativeValid == parent # "absent" /\ (metadata => parent = "value")
BodyValid == replacement \/ (parent = "value" /\ selected)
Valid == NativeValid /\ BodyValid
Prepare == /\ phase = "ready"
           /\ phase' = "done"
           /\ accepted' = (CASE Policy = "native-body-always" -> NativeValid /\ parent = "value" /\ selected
                            [] Policy = "formatter-bypass" -> replacement \/ Valid
                            [] OTHER -> Valid)
           /\ tagFieldRead' = (tagged /\ (Policy = "nil-tag-read" \/ parent = "value"))
           /\ metadataFieldRead' = (accepted' /\ metadata)
           /\ UNCHANGED <<parent, selected, replacement, metadata, tagged>>
Spec == Init /\ [][Prepare]_vars
NativeTagSafety == tagFieldRead => parent = "value"
MetadataSafety == metadataFieldRead => parent = "value"
CheckedPublication == accepted => Valid
ValidReplacement == phase = "done" /\ Valid => accepted
=============================================================================
