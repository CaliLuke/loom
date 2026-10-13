-------------------- MODULE ClientInterceptorDispatch --------------------
EXTENDS Naturals, Sequences, TLC
CONSTANT Repair
VARIABLES phase, kind, secondary, chain, enabled, failed,
          opening, messages, asserted, captured, closed
vars == <<phase, kind, secondary, chain, enabled, failed,
          opening, messages, asserted, captured, closed>>
Profiles == {"stream", "combined", "payload", "generic"}
StreamRole == kind = "stream" \/ kind = "final" \/ (kind = "mixed" /\ secondary)
StreamAccess(p) == p \in {"stream", "combined"}
OpenAccess(p) == p # "stream"
Selected == IF enabled THEN chain ELSE <<>>
ExpectedOpening == SelectSeq(Selected, LAMBDA p: OpenAccess(p))
ExpectedMessages == IF StreamRole /\ ~failed
                    THEN SelectSeq(Selected, LAMBDA p: StreamAccess(p)) ELSE <<>>
Init == /\ phase = "choose" /\ kind = "unset" /\ secondary = FALSE
        /\ chain = <<>> /\ enabled = FALSE /\ failed = FALSE
        /\ opening = <<>> /\ messages = <<>> /\ asserted = FALSE
        /\ captured = "original" /\ closed = "delegate"
Choose == /\ phase = "choose"
          /\ kind' \in {"mixed", "stream", "final", "unary"}
          /\ secondary' \in (IF kind' = "mixed" THEN BOOLEAN ELSE {FALSE})
          /\ enabled' \in BOOLEAN /\ failed' \in BOOLEAN
          /\ chain' \in (IF kind' = "unary" THEN {"payload", "generic"} \X {"payload", "generic"}
                         ELSE Profiles \X Profiles)
          /\ phase' = "dispatch"
          /\ UNCHANGED <<opening, messages, asserted, captured, closed>>
Dispatch == /\ phase = "dispatch"
            /\ opening' = IF Repair = "skip-ordinary" /\ ~StreamRole
                          THEN <<>> ELSE ExpectedOpening
            /\ asserted' = (Repair # "ordinary-only" /\ ~failed /\ Len(SelectSeq(Selected, LAMBDA p: StreamAccess(p))) > 0
                            /\ (Repair = "current" \/ StreamRole))
            /\ messages' = IF Repair = "ordinary-only" THEN <<>> ELSE ExpectedMessages
            /\ phase' = "done"
            /\ UNCHANGED <<kind, secondary, chain, enabled, failed, captured, closed>>
Next == Choose \/ Dispatch \/ (phase = "done" /\ UNCHANGED vars)
OrdinaryResultRetained == phase # "done" \/ StreamRole \/ ~asserted
OpeningCallbacks == phase # "done" \/ opening = ExpectedOpening
MessageCallbacks == phase # "done" \/ messages = ExpectedMessages
FailedOpenNotWrapped == phase # "done" \/ ~failed \/ (~asserted /\ messages = <<>>)
ContextAndClose == captured = "original" /\ closed = "delegate"
=============================================================================
