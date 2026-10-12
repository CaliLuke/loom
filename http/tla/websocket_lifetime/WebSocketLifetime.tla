----------------------- MODULE WebSocketLifetime -----------------------
EXTENDS Naturals, Sequences, TLC

CONSTANTS Design, Witness, MessageMax, BufferMax, MessageCountMax
VARIABLES socket, terminal, canceled, watcher, attached, peerGone, dispatch,
          wire, inbox, flight, decoding, delivered, admitted, halfClosed,
          decodeFailures, closeWriteFailed, cause, eventDone

vars == <<socket, terminal, canceled, watcher, attached, peerGone, dispatch,
          wire, inbox, flight, decoding, delivered, admitted, halfClosed,
          decodeFailures, closeWriteFailed, cause, eventDone>>

Frames == {1, 2, 3, 5, 6, 7}
Size(f) == IF f \in {5, 6, 7} THEN 0 ELSE IF f = 2 THEN MessageMax ELSE 1
RECURSIVE Bytes(_)
Bytes(q) == IF q = <<>> THEN 0 ELSE Size(Head(q)) + Bytes(Tail(q))
DecodeBytes == IF decoding = 0 THEN 0 ELSE Size(decoding)
BufferedBytes == Bytes(inbox) + DecodeBytes
FlightBytes == IF flight = 0 THEN 0 ELSE Size(flight)
BufferedMessages == Len(inbox) + IF decoding = 0 THEN 0 ELSE 1
CountFits == Design = "bytes_only" \/ BufferedMessages < MessageCountMax

Init == /\ socket = TRUE /\ terminal = FALSE /\ canceled = FALSE
        /\ watcher = TRUE /\ attached = TRUE /\ peerGone = FALSE
        /\ dispatch = TRUE
        /\ wire = CASE Witness = "traffic" -> <<1, 2, 3, 4>>
                       [] Witness = "empty" -> <<5, 6, 7, 4>>
                       [] OTHER -> <<>>
        /\ inbox = <<>> /\ flight = 0 /\ decoding = 0
        /\ delivered = <<>> /\ admitted = <<>> /\ halfClosed = FALSE
        /\ decodeFailures = 0 /\ closeWriteFailed = FALSE
        /\ cause = "none" /\ eventDone = FALSE

Stop(why) == /\ terminal' = TRUE /\ socket' = FALSE
             /\ canceled' = TRUE /\ watcher' = FALSE
             /\ cause' = IF cause = "none" THEN why ELSE cause

Fault == /\ ~eventDone /\ Witness \in {"close", "read", "return"}
         /\ eventDone' = TRUE
         /\ closeWriteFailed' = (Witness = "close")
         /\ IF Design = "old"
               THEN /\ terminal' = TRUE
                    /\ cause' = Witness
                    /\ UNCHANGED <<socket, canceled, watcher>>
               ELSE Stop(Witness)
         /\ UNCHANGED <<attached, peerGone, dispatch, wire, inbox, flight,
                        decoding, delivered, admitted, halfClosed, decodeFailures>>

Disconnect == /\ Witness = "disconnect" /\ ~peerGone
              /\ peerGone' = TRUE
              /\ UNCHANGED <<socket, terminal, canceled, watcher, attached,
                             dispatch, wire, inbox, flight, decoding, delivered,
                             admitted, halfClosed, decodeFailures,
                             closeWriteFailed, cause, eventDone>>

DetectDisconnect == /\ peerGone /\ ~terminal
                    /\ (Design # "old" \/ ~dispatch)
                    /\ Stop("peer")
                    /\ UNCHANGED <<attached, peerGone, dispatch, wire, inbox,
                                   flight, decoding, delivered, admitted,
                                   halfClosed, decodeFailures,
                                   closeWriteFailed, eventDone>>

ReadFrame == /\ Witness \in {"traffic", "empty"} /\ ~terminal /\ flight = 0
             /\ wire # <<>> /\ Head(wire) \in Frames
             /\ flight' = Head(wire) /\ wire' = Tail(wire)
             /\ UNCHANGED <<socket, terminal, canceled, watcher, attached,
                            peerGone, dispatch, inbox, decoding, delivered,
                            admitted, halfClosed, decodeFailures,
                            closeWriteFailed, cause, eventDone>>

QueueFrame == /\ flight # 0 /\ ~terminal
              /\ BufferedBytes + Size(flight) <= BufferMax
              /\ CountFits
              /\ inbox' = Append(inbox, flight)
              /\ admitted' = Append(admitted, flight) /\ flight' = 0
              /\ UNCHANGED <<socket, terminal, canceled, watcher, attached,
                             peerGone, dispatch, wire, decoding, delivered,
                             halfClosed, decodeFailures, closeWriteFailed,
                             cause, eventDone>>

Overload == /\ Design \in {"overload", "bytes_only"} /\ flight # 0 /\ ~terminal
            /\ (BufferedBytes + Size(flight) > BufferMax \/ ~CountFits)
            /\ Stop("overload")
            /\ UNCHANGED <<attached, peerGone, dispatch, wire, inbox, flight,
                           decoding, delivered, admitted, halfClosed,
                           decodeFailures, closeWriteFailed, eventDone>>

ReadPeerClose == /\ Witness \in {"traffic", "empty"} /\ ~terminal /\ flight = 0
                 /\ wire = <<4>>
                 /\ wire' = <<>> /\ peerGone' = TRUE
                 /\ Stop("peer")
                 /\ UNCHANGED <<attached, dispatch, inbox, flight, decoding,
                                delivered, admitted, halfClosed,
                                decodeFailures, closeWriteFailed, eventDone>>

StartDecode == /\ Witness \in {"traffic", "empty"} /\ ~terminal /\ ~dispatch
               /\ decoding = 0 /\ inbox # <<>>
               /\ decoding' = Head(inbox) /\ inbox' = Tail(inbox)
               /\ UNCHANGED <<socket, terminal, canceled, watcher, attached,
                              peerGone, dispatch, wire, flight, delivered,
                              admitted, halfClosed, decodeFailures,
                              closeWriteFailed, cause, eventDone>>

FinishDecode == /\ decoding # 0 /\ ~terminal
                /\ delivered' = Append(delivered, decoding) /\ decoding' = 0
                /\ UNCHANGED <<socket, terminal, canceled, watcher, attached,
                               peerGone, dispatch, wire, inbox, flight,
                               admitted, halfClosed, decodeFailures,
                               closeWriteFailed, cause, eventDone>>

ConsumerProgress == /\ Witness \in {"traffic", "empty"} /\ dispatch
                    /\ dispatch' = FALSE
                    /\ UNCHANGED <<socket, terminal, canceled, watcher,
                                   attached, peerGone, wire, inbox, flight,
                                   decoding, delivered, admitted, halfClosed,
                                   decodeFailures, closeWriteFailed, cause,
                                   eventDone>>

HalfClose == /\ Witness = "half" /\ ~halfClosed /\ ~terminal
             /\ halfClosed' = TRUE
             /\ UNCHANGED <<socket, terminal, canceled, watcher, attached,
                            peerGone, dispatch, wire, inbox, flight, decoding,
                            delivered, admitted, decodeFailures,
                            closeWriteFailed, cause, eventDone>>

DecodeFailure == /\ Witness = "decode" /\ decodeFailures = 0 /\ ~terminal
                 /\ decodeFailures' = 1
                 /\ UNCHANGED <<socket, terminal, canceled, watcher, attached,
                                peerGone, dispatch, wire, inbox, flight,
                                decoding, delivered, admitted, halfClosed,
                                closeWriteFailed, cause, eventDone>>

LocalCancel == /\ Witness = "late" /\ ~terminal
               /\ Stop("cancel") /\ attached' = FALSE
               /\ UNCHANGED <<peerGone, dispatch, wire, inbox, flight,
                              decoding, delivered, admitted, halfClosed,
                              decodeFailures, closeWriteFailed, eventDone>>

LateAttach == /\ Witness = "late" /\ terminal /\ ~attached
              /\ attached' = TRUE /\ socket' = FALSE
              /\ UNCHANGED <<terminal, canceled, watcher, peerGone, dispatch,
                             wire, inbox, flight, decoding, delivered,
                             admitted, halfClosed, decodeFailures,
                             closeWriteFailed, cause, eventDone>>

Next == Fault \/ Disconnect \/ DetectDisconnect \/ ReadFrame \/ QueueFrame
        \/ Overload \/ ReadPeerClose \/ StartDecode \/ FinishDecode
        \/ ConsumerProgress \/ HalfClose \/ DecodeFailure \/ LocalCancel \/ LateAttach

Spec == Init /\ [][Next]_vars
        /\ WF_vars(DetectDisconnect) /\ WF_vars(ReadFrame)
        /\ WF_vars(QueueFrame) /\ WF_vars(Overload) /\ WF_vars(ReadPeerClose)
        /\ WF_vars(StartDecode) /\ WF_vars(FinishDecode)

TerminalReleases == terminal => ~socket
TerminalStopsOwnedWork == terminal => ~watcher /\ canceled
HalfClosePreservesOutput == halfClosed => socket /\ ~canceled
DecodeFailureRecoverable == decodeFailures > 0 => socket /\ ~terminal
BoundedRawBytes == BufferedBytes <= BufferMax
                   /\ FlightBytes <= MessageMax
                   /\ BufferedBytes + FlightBytes <= BufferMax + MessageMax
BoundedMessages == BufferedMessages <= MessageCountMax
FIFO == delivered = SubSeq(admitted, 1, Len(delivered))
NoSilentLoss == ~terminal => admitted = delivered \o
                (IF decoding = 0 THEN <<>> ELSE <<decoding>>) \o inbox
DisconnectCancels == peerGone ~> canceled
TrafficEventuallyStops == <>terminal
=============================================================================
