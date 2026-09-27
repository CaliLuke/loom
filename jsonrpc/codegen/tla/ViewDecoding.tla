--------------------------- MODULE ViewDecoding ---------------------------
EXTENDS Naturals, Sequences
CONSTANT CheckMarker
VARIABLES frame, fixed, phase, selected, accepted
vars == <<frame, fixed, phase, selected, accepted>>
Views == {"default", "tiny"}
Fields(view) == IF view = "default" THEN {"id", "name"} ELSE {"id"}
Init == /\ frame \in [marker: Views \cup {"", "unknown", "non-string"}, body: {{"id"}, {"id", "name"}, {}}]
        /\ fixed \in Views \cup {""}
        /\ phase = "receive" /\ selected = "" /\ accepted = FALSE
Receive == /\ phase = "receive"
           /\ LET fallback == IF fixed = "" THEN "default" ELSE fixed
                  chosen == IF frame.marker = "" THEN fallback ELSE frame.marker
              IN /\ selected' = IF CheckMarker THEN chosen ELSE fallback
                 /\ accepted' = IF CheckMarker
                      THEN chosen \in Views /\ (fixed = "" \/ chosen = fixed)
                           /\ Fields(chosen) \subseteq frame.body
                      ELSE TRUE
           /\ phase' = "done"
           /\ UNCHANGED <<frame, fixed>>
Spec == Init /\ [][Receive]_vars
DeclaredViewOnly == phase = "done" /\ accepted => selected \in Views
FixedViewPreserved == phase = "done" /\ accepted /\ fixed # "" => selected = fixed
MarkerPreserved == phase = "done" /\ accepted /\ frame.marker # "" => selected = frame.marker
RequiredFieldsPresent == phase = "done" /\ accepted => Fields(selected) \subseteq frame.body
LegacyFallback == phase = "done" /\ frame.marker = "" => selected = IF fixed = "" THEN "default" ELSE fixed
=============================================================================
