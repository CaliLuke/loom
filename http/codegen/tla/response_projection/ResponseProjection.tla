----------------------- MODULE ResponseProjection -----------------------
EXTENDS Naturals, FiniteSets
CONSTANT Repair
VARIABLES phase, channel, shape, metadata, hasID, view, wireView, wire, calls, emitted
vars == <<phase, channel, shape, metadata, hasID, view, wireView, wire, calls, emitted>>
Channels == {"rpc-http", "rpc-ws", "http-ws", "mixed-sse"}
Shapes == {"scalar", "constructed", "whole"}
Raw == {"value", "secret"}
Selected == IF shape = "whole" THEN Raw
            ELSE IF shape = "constructed" /\ view = "default" THEN {"value", "public"}
            ELSE {"value"}
Expected == IF channel = "mixed-sse" THEN {"event"} ELSE Selected
Capture == channel = "rpc-http" /\ metadata
CurrentProjection == IF channel = "mixed-sse" THEN {"event"}
                     ELSE IF Capture \/ shape = "constructed" THEN Selected
                     ELSE Raw
ExpectedCalls == IF channel = "rpc-http" THEN (IF metadata THEN 2 ELSE 1) ELSE 0
Init == /\ phase = "choose"
        /\ channel = "unset" /\ shape = "unset"
        /\ metadata = FALSE /\ hasID = TRUE /\ view = "default" /\ wireView = "unset"
        /\ wire = {} /\ calls = 0 /\ emitted = FALSE
Choose == /\ phase = "choose"
          /\ \E c \in Channels, s \in Shapes, m \in BOOLEAN, i \in BOOLEAN, v \in {"default", "tiny"}:
               /\ channel' = c /\ shape' = s /\ metadata' = m /\ hasID' = i /\ view' = v
               /\ (m => c = "rpc-http")
               /\ (Repair = "uniform-bodies" => ~m)
          /\ phase' = "frame"
          /\ UNCHANGED <<wireView, wire, calls, emitted>>
Frame == /\ phase = "frame"
         /\ emitted' = (channel # "rpc-http" \/ hasID)
         /\ wireView' = view
         /\ wire' = IF channel = "rpc-http" /\ ~hasID THEN {}
                    ELSE IF Repair = "current" THEN CurrentProjection
                    ELSE IF Repair = "scalar-patch" THEN
                      (IF channel = "rpc-http" /\ shape = "scalar" THEN Selected ELSE CurrentProjection)
                    ELSE IF Repair = "unary-everywhere" THEN Selected
                    ELSE IF Repair = "first-view" /\ shape = "constructed" /\ channel # "mixed-sse" THEN {"value", "public"}
                    ELSE Expected
         /\ calls' = IF channel = "rpc-http" /\ ~hasID THEN 0
                     ELSE IF Repair = "capture-everywhere" /\ channel = "rpc-http" THEN 2
                     ELSE ExpectedCalls
         /\ phase' = "done"
         /\ UNCHANGED <<channel, shape, metadata, hasID, view>>
Next == Choose \/ Frame
SelectedWire == phase # "done" \/ ~emitted \/ wire = Expected
NoExcludedFields == phase # "done" \/ ~emitted \/ "secret" \notin (wire \ Expected)
NotificationSuppressed == phase # "done" \/ channel # "rpc-http" \/ hasID \/ (~emitted /\ wire = {} /\ calls = 0)
CodecBoundaryPreserved == phase # "done" \/ ~emitted \/ calls = ExpectedCalls
ViewCorrespondence == phase # "done" \/ ~emitted \/ wireView = view
=============================================================================
