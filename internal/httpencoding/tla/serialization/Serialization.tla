------------------------- MODULE Serialization -------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Policy
VARIABLES shape, location, authored, phase, accepted, client, server, documented, carrier
vars == <<shape, location, authored, phase, accepted, client, server, documented, carrier>>
Shapes == {"scalar", "sequence", "map", "object", "nested"}
Locations == {"query", "form"}
Styles == {"default", "form", "formTrue", "formFalse", "deep", "deepTrue", "deepFalse", "json"}
Target == CASE shape = "scalar" -> "scalar"
            [] shape = "sequence" -> "repeat"
            [] location = "query" /\ shape = "object" -> "unsupported"
            [] OTHER -> "json"
LegacyRuntime == CASE Target \in {"scalar", "repeat", "unsupported"} -> Target
                   [] shape = "nested" -> "recursive-brackets"
                   [] OTHER -> "brackets"
DefaultSource == CASE shape = "scalar" -> "scalar"
                   [] shape = "sequence" -> "repeat"
                   [] location = "form" -> "json"
                   [] OTHER -> "flat"
Source == CASE authored = "json" -> "json"
            [] authored \in {"deep", "deepTrue", "deepFalse"} ->
                 IF shape \in {"map", "object"} THEN "brackets" ELSE "undefined"
            [] authored = "default" /\ location = "form" /\ shape \in {"map", "object", "nested"} -> "json"
            [] shape = "scalar" -> "scalar"
            [] shape = "sequence" -> IF authored = "formFalse" THEN "comma" ELSE "repeat"
            [] OTHER -> "flat"
KnownEquivalent == Target # "unsupported" /\ Source = Target
Init == /\ shape \in Shapes /\ location \in Locations /\ authored \in Styles
        /\ phase = "unplanned" /\ accepted = FALSE
        /\ client = "none" /\ server = "none" /\ documented = "none" /\ carrier = "none"
Plan == /\ phase = "unplanned"
        /\ accepted' = IF Policy = "legacy" THEN authored \in {"default", "form"} ELSE KnownEquivalent
        /\ client' = IF Policy \in {"legacy", "metadata_only"} THEN LegacyRuntime ELSE Target
        /\ server' = IF Policy \in {"checked", "native_json"} THEN Target ELSE LegacyRuntime
        /\ documented' = IF Policy = "legacy" THEN DefaultSource ELSE Target
        /\ carrier' = IF Policy = "checked" /\ Target = "json" THEN "json_presence" ELSE "native"
        /\ phase' = "planned"
        /\ UNCHANGED <<shape, location, authored>>
Next == Plan
Spec == Init /\ [][Next]_vars
ImportPreservesWire == phase = "planned" /\ accepted => KnownEquivalent
ImportKeepsEquivalent == phase = "planned" /\ KnownEquivalent => accepted
ExportPreservesWire == phase = "planned" /\ Target # "unsupported" => documented = client
ClientServerAgree == phase = "planned" => client = server
UniformCompositeMode == phase = "planned" /\ Target = "json" => client = "json" /\ server = "json"
JSONCarrierRejectsNull == phase = "planned" /\ Target = "json" => carrier = "json_presence"
=============================================================================
