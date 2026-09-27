--------------------------- MODULE StreamingViews ---------------------------
EXTENDS Naturals, Sequences
CONSTANT CarryView
VARIABLES phase, selected, wire, decoded, accepted
vars == <<phase, selected, wire, decoded, accepted>>
Views == {"default", "tiny"}
Fields(view) == IF view = "default" THEN {"id", "name"} ELSE {"id"}
Init == /\ phase = "send" /\ selected = <<>> /\ wire = <<>>
        /\ decoded = <<>> /\ accepted = <<>>
Send(view) == /\ phase = "send" /\ Len(wire) < 2 /\ view \in Views
              /\ selected' = Append(selected, view)
              /\ wire' = Append(wire, [body |-> Fields(view), marker |-> IF CarryView THEN view ELSE ""])
              /\ phase' = IF Len(wire') = 2 THEN "receive" ELSE phase
              /\ UNCHANGED <<decoded, accepted>>
Receive == /\ phase = "receive" /\ Len(decoded) < 2
           /\ LET message == wire[Len(decoded) + 1]
                  view == IF message.marker = "" THEN "default" ELSE message.marker
              IN /\ decoded' = Append(decoded, view)
                 /\ accepted' = Append(accepted, IF CarryView THEN Fields(view) \subseteq message.body ELSE TRUE)
           /\ phase' = IF Len(decoded') = 2 THEN "done" ELSE phase
           /\ UNCHANGED <<selected, wire>>
Next == (\E view \in Views: Send(view)) \/ Receive
Spec == Init /\ [][Next]_vars
ValidFramesAccepted == \A i \in 1..Len(accepted): accepted[i]
PerFrameViewPreserved == \A i \in 1..Len(decoded): decoded[i] = selected[i]
PayloadShapePreserved == \A i \in 1..Len(wire): wire[i].body = Fields(selected[i])
=============================================================================
