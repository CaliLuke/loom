-------------------------- MODULE ViewBody --------------------------
EXTENDS Naturals, Sequences
CONSTANTS UseSelectedBody, OmitMissingMappedData
VARIABLES mode, mapping, phase, request, validValue, selected, projected, wire, sent
vars == <<mode, mapping, phase, request, validValue, selected, projected, wire, sent>>
Views == {"default", "full", "tiny"}
Fields(view) == IF view = "full" THEN {"id", "name", "secret"}
                ELSE IF view = "tiny" THEN {"id"} ELSE {"id", "name"}
Resolve(selectionMode, selection) ==
    IF selectionMode = "fixed" THEN "full"
    ELSE IF selectionMode = "discriminator" THEN
        CASE selection = "legacy" -> "default"
          [] selection = "updated" -> "full"
          [] OTHER -> "invalid"
    ELSE IF selection = "" THEN "default" ELSE selection
EmptyData(kind) == IF kind = "text" THEN "empty" ELSE "null"
Init == /\ mode \in {"dynamic", "fixed", "discriminator"}
        /\ mapping \in IF mode = "discriminator" THEN {"whole"} ELSE {"whole", "text", "json"}
        /\ phase = "choose"
        /\ request = ""
        /\ validValue = TRUE
        /\ selected = ""
        /\ projected = {}
        /\ wire = {}
        /\ sent = <<>>
Choose == /\ phase = "choose"
          /\ request' \in {"", "default", "full", "tiny", "missing", "legacy", "updated"}
          /\ validValue' \in BOOLEAN
          /\ phase' = "project"
          /\ UNCHANGED <<mode, mapping, selected, projected, wire, sent>>
Project == LET view == Resolve(mode, request) IN
           /\ phase = "project"
           /\ IF validValue /\ view \in Views
                 THEN /\ selected' = view
                      /\ projected' = Fields(view)
                      /\ phase' = "encode"
                 ELSE /\ phase' = "rejected"
                      /\ UNCHANGED <<selected, projected>>
           /\ UNCHANGED <<mode, mapping, request, validValue, wire, sent>>
Encode == LET bodyView == IF UseSelectedBody \/ mode # "dynamic"
                         THEN selected ELSE "default"
              converted == projected \cap Fields(bodyView)
              data == IF mapping = "whole" THEN "object"
                      ELSE IF "name" \in Fields(bodyView)
                           THEN IF "name" \in converted THEN "value" ELSE EmptyData(mapping)
                           ELSE IF OmitMissingMappedData THEN EmptyData(mapping) ELSE "invalid-read"
          IN
          /\ phase = "encode"
          /\ wire' = converted
          /\ sent' = Append(sent, [view |-> selected, fields |-> wire', data |-> data])
          /\ phase' = "written"
          /\ UNCHANGED <<mode, mapping, request, validValue, selected, projected>>
Again == /\ phase \in {"written", "rejected"}
         /\ Len(sent) < 2
         /\ phase' = "choose"
         /\ selected' = ""
         /\ projected' = {}
         /\ wire' = {}
         /\ UNCHANGED <<mode, mapping, request, validValue, sent>>
Next == Choose \/ Project \/ Encode \/ Again
Spec == Init /\ [][Next]_vars
SelectedBody == mapping = "whole" =>
                  \A i \in 1..Len(sent): sent[i].fields = Fields(sent[i].view)
MappedData == mapping # "whole" =>
              \A i \in 1..Len(sent): sent[i].data =
                  IF "name" \in Fields(sent[i].view) THEN "value" ELSE EmptyData(mapping)
WellFormedBodyAccess == \A i \in 1..Len(sent): sent[i].data # "invalid-read"
NoRejectedWrite == phase = "rejected" => wire = {}
=============================================================================
