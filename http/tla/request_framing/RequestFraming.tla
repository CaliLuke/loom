----------------------- MODULE RequestFraming -----------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
VARIABLES phase, reader, closeFails, readResult, closeState,
          closeOwner, closeCalls, cancelled, errors, noBody
vars == <<phase, reader, closeFails, readResult, closeState,
          closeOwner, closeCalls, cancelled, errors, noBody>>
Init == /\ phase = "new"
        /\ reader \in {"empty", "data", "error", "blocked"}
        /\ closeFails \in BOOLEAN
        /\ readResult = "pending"
        /\ closeState = "open"
        /\ closeOwner = "none"
        /\ closeCalls = 0
        /\ cancelled = FALSE
        /\ errors = {}
        /\ noBody = FALSE
Start == /\ phase = "new"
         /\ phase' = "reading"
         /\ UNCHANGED <<reader, closeFails, readResult, closeState,
                         closeOwner, closeCalls, cancelled, errors, noBody>>
Read == /\ phase = "reading" /\ readResult = "pending"
        /\ (reader # "blocked" \/ closeState # "open")
        /\ readResult' = CASE reader = "empty" -> "eof"
                            [] reader = "data" -> "data"
                            [] OTHER -> "error"
        /\ UNCHANGED <<phase, reader, closeFails, closeState, closeOwner,
                        closeCalls, cancelled, errors, noBody>>
Cancel == /\ phase \in {"new", "reading", "prepared"} /\ ~cancelled
          /\ cancelled' = TRUE
          /\ phase' = IF phase = "prepared" THEN "failed" ELSE phase
          /\ errors' = IF phase = "prepared" THEN errors \cup {"context"} ELSE errors
          /\ UNCHANGED <<reader, closeFails, readResult, closeState,
                          closeOwner, closeCalls, noBody>>
ClaimClose == /\ closeState = "open"
              /\ (cancelled \/ readResult # "pending")
              /\ closeState' = "closing"
              /\ closeOwner' = IF cancelled THEN "cancel" ELSE "probe"
              /\ closeCalls' = closeCalls + 1
              /\ UNCHANGED <<phase, reader, closeFails, readResult,
                              cancelled, errors, noBody>>
FinishClose == /\ closeState = "closing"
               /\ closeState' = "closed"
               /\ UNCHANGED <<phase, reader, closeFails, readResult, closeOwner,
                               closeCalls, cancelled, errors, noBody>>
Causes == (IF cancelled THEN {"context"} ELSE {})
          \cup (IF closeFails THEN {"close"} ELSE {})
          \cup (CASE readResult = "error" -> {"read"}
                   [] readResult = "data" -> {"length"}
                   [] OTHER -> {})
Finalize == /\ phase = "reading" /\ readResult # "pending"
            /\ closeState = "closed"
            /\ LET causes == IF Mode = "lost_close" THEN Causes \ {"close"} ELSE Causes
               IN /\ errors' = causes
                  /\ phase' = IF causes = {} THEN "prepared" ELSE "failed"
            /\ noBody' = (readResult = "eof")
            /\ UNCHANGED <<reader, closeFails, readResult, closeState,
                            closeOwner, closeCalls, cancelled>>
Publish == /\ phase = "prepared" /\ ~cancelled
           /\ phase' = "published"
           /\ UNCHANGED <<reader, closeFails, readResult, closeState, closeOwner,
                           closeCalls, cancelled, errors, noBody>>
CurrentPublish == /\ Mode = "current" /\ phase = "reading"
                  /\ phase' = "published"
                  /\ UNCHANGED <<reader, closeFails, readResult, closeState,
                                  closeOwner, closeCalls, cancelled, errors, noBody>>
DoubleClose == /\ Mode = "double_close" /\ closeState = "closed"
               /\ closeOwner = "cancel" /\ readResult # "pending" /\ closeCalls = 1
               /\ closeCalls' = 2
               /\ UNCHANGED <<phase, reader, closeFails, readResult, closeState,
                               closeOwner, cancelled, errors, noBody>>
Next == Start \/ Read \/ Cancel \/ ClaimClose \/ FinishClose \/ Finalize
        \/ Publish \/ CurrentPublish \/ DoubleClose
Spec == Init /\ [][Next]_vars
        /\ WF_vars(Start) /\ WF_vars(Read) /\ WF_vars(ClaimClose)
        /\ WF_vars(FinishClose) /\ WF_vars(Finalize)
CloseOnce == closeCalls <= 1
PublicationSafe == phase = "published" =>
  (noBody /\ readResult = "eof" /\ closeState = "closed" /\ ~cancelled /\ ~closeFails)
EmptyProven == noBody => readResult = "eof"
FailuresRetained == phase = "failed" => Causes \subseteq errors
CancelReleased == cancelled ~> (closeState = "closed" /\ phase = "failed")
====================================================================
