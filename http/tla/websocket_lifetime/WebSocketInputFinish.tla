----------------------- MODULE WebSocketInputFinish -----------------------
EXTENDS Naturals, Sequences
CONSTANTS Mode, Peer
VARIABLES wire, queued, client, socket, canceled, consumed, service, timedOut
vars == <<wire, queued, client, socket, canceled, consumed, service, timedOut>>
Init == /\ wire = <<"data", "null">> /\ queued = <<>>
        /\ client = "sent" /\ socket = TRUE /\ canceled = FALSE
        /\ consumed = <<>> /\ service = (IF Peer = "final" THEN "closing" ELSE "running")
        /\ timedOut = FALSE
Pump == /\ socket /\ Len(wire) > 0
        /\ queued' = Append(queued, Head(wire)) /\ wire' = Tail(wire)
        /\ UNCHANGED <<client, socket, canceled, consumed, service, timedOut>>
Read == /\ service = "running" /\ ~canceled /\ Len(queued) > 0
        /\ consumed' = Append(consumed, Head(queued))
        /\ service' = IF Head(queued) = "null" THEN "finished" ELSE service
        /\ queued' = Tail(queued)
        /\ UNCHANGED <<wire, client, socket, canceled, timedOut>>
ImmediateClose == /\ Mode = "immediate" /\ client = "sent"
                  /\ client' = "done" /\ socket' = FALSE
                  /\ UNCHANGED <<wire, queued, canceled, consumed, service, timedOut>>
DetectDisconnect == /\ ~socket /\ ~canceled /\ Len(wire) = 0
                    /\ canceled' = TRUE
                    /\ UNCHANGED <<wire, queued, client, socket, consumed, service, timedOut>>
FlushBeforeEOF == /\ ~socket /\ Len(wire) > 0
                  /\ queued' = Append(queued, Head(wire)) /\ wire' = Tail(wire)
                  /\ UNCHANGED <<client, socket, canceled, consumed, service, timedOut>>
PeerFinishes == /\ service = "finished" /\ socket
                /\ socket' = FALSE /\ client' = "done"
                /\ UNCHANGED <<wire, queued, canceled, consumed, service, timedOut>>
Timeout == /\ Mode = "bounded" /\ client = "sent" /\ service # "finished"
           /\ timedOut' = TRUE /\ socket' = FALSE /\ client' = "done"
           /\ UNCHANGED <<wire, queued, canceled, consumed, service>>
AbruptPeer == /\ Peer = "abrupt" /\ socket
              /\ socket' = FALSE /\ client' = "done"
              /\ UNCHANGED <<wire, queued, canceled, consumed, service, timedOut>>
Next == AbruptPeer \/ Pump \/ Read \/ ImmediateClose \/ DetectDisconnect \/ FlushBeforeEOF \/ PeerFinishes \/ Timeout
Spec == Init /\ [][Next]_vars /\ WF_vars(Pump) /\ WF_vars(Read)
        /\ WF_vars(ImmediateClose) /\ WF_vars(DetectDisconnect)
        /\ WF_vars(AbruptPeer) /\ WF_vars(FlushBeforeEOF) /\ WF_vars(PeerFinishes) /\ WF_vars(Timeout)
OrderlyInputPreserved == Peer = "handler" /\ canceled /\ ~timedOut => consumed = <<"data", "null">>
CloseEventuallyReturns == <>(client = "done")
FIFO == consumed \in {<<>>, <<"data">>, <<"data", "null">>}
=============================================================================
