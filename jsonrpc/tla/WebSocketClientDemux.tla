------------------------ MODULE WebSocketClientDemux ------------------------
(***************************************************************************)
(* How a generated JSON-RPC WebSocket client routes responses to the       *)
(* streams that share its connection (issue #399).                         *)
(*                                                                         *)
(* Every stream opened on one client uses the client's connection. A       *)
(* stream registers each request under its JSON-RPC id, writes it, and     *)
(* Recv waits for the response to the oldest request. The server answers   *)
(* the written requests in any order and can send an id-less               *)
(* notification. The socket can fail, the client can close the             *)
(* connection, and a stream can be closed or have its context canceled.    *)
(*                                                                         *)
(* DESIGN selects how responses reach the streams:                         *)
(*   "asis"       main before #399: every stream runs its own reader on    *)
(*                the connection, numbers its requests from 1 and looks    *)
(*                responses up in its own pending map. Closing a stream    *)
(*                closes the connection.                                   *)
(*   "one_reader" one reader per connection and one shared pending map,    *)
(*                but each stream still numbers its requests from 1.       *)
(*   "unguarded"  one reader and connection-wide ids, but registration     *)
(*                does not check, under the lock the reader takes to fail  *)
(*                the waiters, that the connection is still usable.        *)
(*   "demux_split" "demux", but the last release drops its reference     *)
(*                under the lock and marks and closes the connection in a  *)
(*                later step, as the first implementation of Release did.  *)
(*   "demux"      one reader, connection-wide ids, registration refused    *)
(*                once the connection is marked closed under that lock,    *)
(*                and the connection marked closed in the same step that   *)
(*                drops its last reference.                                *)
(* See README.md for the mapping to the Go code.                           *)
(***************************************************************************)
EXTENDS Naturals, Sequences, FiniteSets

CONSTANTS
    DESIGN,       \* "asis", "one_reader", "unguarded", "demux_split" or "demux"
    Streams,      \* stream names (strings)
    MaxSends,     \* requests each stream sends
    TIMEOUT,      \* a waiting Recv can time out
    CANCEL,       \* a waiting Recv can be canceled (the request is requeued)
    CLOSE,        \* a stream can be closed or have its context canceled
    SOCKET_FAIL,  \* the socket can fail
    CLIENT_CLOSE, \* the client can close the connection
    NOTIFY        \* the server can send one id-less notification

ASSUME DESIGN \in {"asis", "one_reader", "unguarded", "demux_split", "demux"}

\* "demux_split" is "demux" with a last release that marks and closes the
\* connection only after dropping the reference and unlocking.
IsDemux == DESIGN \in {"demux_split", "demux"}

NoId == 0
None == "none"

\* Each stream numbers its requests from 1 in "asis" and "one_reader".
PerStreamIds == DESIGN \in {"asis", "one_reader"}
MaxId == IF PerStreamIds THEN MaxSends ELSE MaxSends * Cardinality(Streams)
Ids == 1..MaxId

\* "asis" runs one reader per stream; the other designs one per connection.
Readers == IF DESIGN = "asis" THEN Streams ELSE {"conn"}

Msgs == [id: Ids \cup {NoId}, from: Streams \cup {None}]

VARIABLES
    st,        \* st[s]: "new", "open" or "closed" (closed or context canceled)
    failed,    \* failed[s]: the stream recorded a connection error
    sent,      \* sent[s]: requests the stream registered
    ctr,       \* ctr[s]: per-stream id counter ("asis", "one_reader")
    gctr,      \* connection id counter (other designs)
    queue,     \* queue[s]: ids of sent requests awaiting Recv, in send order
    wait,      \* wait[s]: id Recv waits for, or NoId
    wq,        \* wq[s]: id registered and not yet written, or NoId
    wlock,     \* holder of the connection write lock, or None
    pending,   \* registrations <<id, stream>> awaiting a response
    reqs,      \* requests <<id, stream>> the server holds, not answered yet
    wire,      \* messages from the server not read yet
    notified,  \* the server sent its notification
    rd,        \* rd[r]: reader state "idle", "reading" or "done"
    cs,        \* socket state: "open" or "closed"
    dead,      \* the connection is marked closed under its lock (Go: its
               \* recorded terminal error). The client and the last release
               \* set it when they close the socket, and the reader sets it
               \* when it fails the waiters.
    refs,      \* open streams using the connection ("demux" closes it at 0)
    delivered, \* results <<stream, id, from>> held for Recv; from is the
               \* stream whose request the server answered, or "err"
    lost,      \* responses dropped while their own stream still waited
    rel        \* "demux_split": "pending" while the last release has dropped
               \* its reference but not closed the connection yet, "bad"
               \* once it closed a connection a stream had acquired since

vars == <<st, failed, sent, ctr, gctr, queue, wait, wq, wlock, pending, reqs,
          wire, notified, rd, cs, dead, refs, delivered, lost, rel>>

Init ==
    /\ st = [s \in Streams |-> "new"]
    /\ failed = [s \in Streams |-> FALSE]
    /\ sent = [s \in Streams |-> 0]
    /\ ctr = [s \in Streams |-> 0]
    /\ gctr = 0
    /\ queue = [s \in Streams |-> <<>>]
    /\ wait = [s \in Streams |-> NoId]
    /\ wq = [s \in Streams |-> NoId]
    /\ wlock = None
    /\ pending = {}
    /\ reqs = {}
    /\ wire = <<>>
    /\ notified = FALSE
    /\ rd = [r \in Readers |-> "idle"]
    /\ cs = "open"
    /\ dead = FALSE
    /\ refs = 0
    /\ delivered = {}
    /\ lost = {}
    /\ rel = "none"

TypeOK ==
    /\ st \in [Streams -> {"new", "open", "closed"}]
    /\ failed \in [Streams -> BOOLEAN]
    /\ sent \in [Streams -> 0..MaxSends]
    /\ ctr \in [Streams -> 0..MaxSends]
    /\ gctr \in 0..MaxId
    /\ wait \in [Streams -> Ids \cup {NoId}]
    /\ wq \in [Streams -> Ids \cup {NoId}]
    /\ wlock \in Streams \cup {None}
    /\ pending \subseteq (Ids \X Streams)
    /\ reqs \subseteq (Ids \X Streams)
    /\ \A i \in DOMAIN wire : wire[i] \in Msgs
    /\ rd \in [Readers -> {"idle", "reading", "done"}]
    /\ cs \in {"open", "closed"}
    /\ dead \in BOOLEAN
    /\ refs \in 0..Cardinality(Streams)
    /\ delivered \subseteq (Streams \X Ids \X (Streams \cup {"err"}))
    /\ rel \in {"none", "pending", "bad"}
\* The registrations of stream s.
Own(s) == {p \in pending : p[2] = s}

\* Fail every registration in regs: its call gets an error result.
FailAll(regs) == {<<p[2], p[1], "err">> : p \in regs}

-----------------------------------------------------------------------------
(* Stream actions *)

(* The client opens a stream on its connection and counts it. *)
Open(s) ==
    /\ st[s] = "new" /\ cs = "open" /\ ~dead
    /\ st' = [st EXCEPT ![s] = "open"]
    /\ refs' = refs + 1
    /\ UNCHANGED <<failed, sent, ctr, gctr, queue, wait, wq, wlock, pending,
                   reqs, wire, notified, rd, cs, dead, delivered, lost, rel>>

(* Send registers the request and queues it for Recv before writing it.
   The stream's own lock makes the closed check and the registration one
   step. "demux" also refuses the registration when the connection is
   marked closed, checked under the connection lock, so no registration can
   follow the reader failing the waiters. *)
Register(s) ==
    /\ st[s] = "open" /\ ~failed[s] /\ sent[s] < MaxSends /\ wq[s] = NoId
    /\ IF IsDemux /\ dead
       THEN /\ failed' = [failed EXCEPT ![s] = TRUE]
            /\ UNCHANGED <<sent, ctr, gctr, queue, wq, pending>>
       ELSE LET id == IF PerStreamIds THEN ctr[s] + 1 ELSE gctr + 1
            IN /\ sent' = [sent EXCEPT ![s] = @ + 1]
               /\ IF PerStreamIds
                  THEN /\ ctr' = [ctr EXCEPT ![s] = id]
                       /\ UNCHANGED gctr
                  ELSE /\ gctr' = id
                       /\ UNCHANGED ctr
               /\ queue' = [queue EXCEPT ![s] = Append(@, id)]
               /\ wq' = [wq EXCEPT ![s] = id]
               \* "one_reader" keys one shared map by id: a second
               \* registration of the same id replaces the first.
               /\ pending' = IF DESIGN = "one_reader"
                             THEN {p \in pending : p[1] # id} \cup {<<id, s>>}
                             ELSE pending \cup {<<id, s>>}
               /\ UNCHANGED failed
    /\ UNCHANGED <<st, wait, wlock, reqs, wire, notified, rd, cs, dead, refs,
                   delivered, lost, rel>>

(* The write lock of the shared WebSocketStream serializes writers. *)
WriteBegin(s) ==
    /\ wq[s] # NoId /\ wlock = None
    /\ wlock' = s
    /\ UNCHANGED <<st, failed, sent, ctr, gctr, queue, wait, wq, pending, reqs,
                   wire, notified, rd, cs, dead, refs, delivered, lost, rel>>

(* The write reaches the server. On a closed socket the write fails (the
   request is unregistered and the stream records the error) or seems to
   succeed although nobody will answer: a read can fail while writes still
   succeed, for example after the server sent a close frame. *)
WriteEnd(s) ==
    /\ wlock = s
    /\ wlock' = None
    /\ wq' = [wq EXCEPT ![s] = NoId]
    /\ IF cs = "open"
       THEN /\ reqs' = reqs \cup {<<wq[s], s>>}
            /\ UNCHANGED <<failed, pending>>
       ELSE /\ \/ /\ pending' = pending \ {<<wq[s], s>>}
                  /\ failed' = [failed EXCEPT ![s] = TRUE]
               \/ UNCHANGED <<failed, pending>>
            /\ UNCHANGED reqs
    /\ UNCHANGED <<st, sent, ctr, gctr, queue, wait, wire, notified, rd, cs,
                   dead, refs, delivered, lost, rel>>

(* Recv takes the oldest request. *)
RecvBegin(s) ==
    /\ st[s] = "open" /\ ~failed[s] /\ wait[s] = NoId /\ queue[s] # <<>>
    /\ wait' = [wait EXCEPT ![s] = Head(queue[s])]
    /\ queue' = [queue EXCEPT ![s] = Tail(@)]
    /\ UNCHANGED <<st, failed, sent, ctr, gctr, wq, wlock, pending, reqs, wire,
                   notified, rd, cs, dead, refs, delivered, lost, rel>>

(* Recv returns the result held for its request. *)
RecvResult(s) ==
    /\ wait[s] # NoId
    /\ \E d \in delivered :
          /\ d[1] = s /\ d[2] = wait[s]
          /\ delivered' = delivered \ {d}
    /\ wait' = [wait EXCEPT ![s] = NoId]
    /\ UNCHANGED <<st, failed, sent, ctr, gctr, queue, wq, wlock, pending, reqs,
                   wire, notified, rd, cs, dead, refs, lost, rel>>

(* Recv returns an error because the stream is done: closed, context
   canceled, or failed. It does not watch the connection itself. *)
RecvDone(s) ==
    /\ wait[s] # NoId
    /\ st[s] = "closed" \/ failed[s]
    /\ wait' = [wait EXCEPT ![s] = NoId]
    /\ UNCHANGED <<st, failed, sent, ctr, gctr, queue, wq, wlock, pending, reqs,
                   wire, notified, rd, cs, dead, refs, delivered, lost, rel>>

(* The request timer fires: the request is forgotten, so a late response
   is an orphan. A result that was already held is dropped with it. *)
RecvTimeout(s) ==
    /\ TIMEOUT /\ wait[s] # NoId
    /\ pending' = pending \ {<<wait[s], s>>}
    /\ delivered' = {d \in delivered : ~(d[1] = s /\ d[2] = wait[s])}
    /\ wait' = [wait EXCEPT ![s] = NoId]
    /\ UNCHANGED <<st, failed, sent, ctr, gctr, queue, wq, wlock, reqs, wire,
                   notified, rd, cs, dead, refs, lost, rel>>

(* The Recv context is canceled: the request goes back to the queue head. *)
RecvCancel(s) ==
    /\ CANCEL /\ wait[s] # NoId
    /\ queue' = [queue EXCEPT ![s] = <<wait[s]>> \o @]
    /\ wait' = [wait EXCEPT ![s] = NoId]
    /\ UNCHANGED <<st, failed, sent, ctr, gctr, wq, wlock, pending, reqs, wire,
                   notified, rd, cs, dead, refs, delivered, lost, rel>>

(* The stream is closed or its context canceled, and drops its
   registrations. "asis": the stream's reader reads with the stream context,
   so the cancellation closes the shared connection, and Close closes it
   too. "demux": the stream releases the connection, which closes when no
   stream uses it. The other designs leave the connection open. *)
Close(s) ==
    /\ CLOSE /\ st[s] = "open"
    /\ LET closes == \/ DESIGN = "asis"
                     \/ DESIGN = "demux" /\ refs = 1
       IN /\ cs' = IF closes THEN "closed" ELSE cs
          /\ dead' = (dead \/ (closes /\ DESIGN = "demux"))
    \* "demux_split" drops the reference under the lock and closes the
    \* connection in a later step (ReleaseClose), as Release did at first.
    /\ rel' = IF DESIGN = "demux_split" /\ refs = 1 THEN "pending" ELSE rel
    /\ st' = [st EXCEPT ![s] = "closed"]
    /\ pending' = pending \ Own(s)
    /\ refs' = refs - 1
    /\ UNCHANGED <<failed, sent, ctr, gctr, queue, wait, wq, wlock, reqs, wire,
                   notified, rd, delivered, lost>>

(* "demux_split": the release that dropped the last reference now closes the
   connection, after unlocking. A stream may have acquired it in between. *)
ReleaseClose ==
    /\ rel = "pending"
    /\ dead' = TRUE
    /\ cs' = "closed"
    /\ rel' = IF refs > 0 THEN "bad" ELSE "none"
    /\ UNCHANGED <<st, failed, sent, ctr, gctr, queue, wait, wq, wlock, pending,
                   reqs, wire, notified, rd, refs, delivered, lost>>

-----------------------------------------------------------------------------
(* Environment *)

(* The server answers a written request, in any order. *)
Respond(r) ==
    /\ cs = "open" /\ r \in reqs
    /\ reqs' = reqs \ {r}
    /\ wire' = Append(wire, [id |-> r[1], from |-> r[2]])
    /\ UNCHANGED <<st, failed, sent, ctr, gctr, queue, wait, wq, wlock, pending,
                   notified, rd, cs, dead, refs, delivered, lost, rel>>

(* The server sends an id-less notification. *)
Notify ==
    /\ NOTIFY /\ cs = "open" /\ ~notified
    /\ notified' = TRUE
    /\ wire' = Append(wire, [id |-> NoId, from |-> None])
    /\ UNCHANGED <<st, failed, sent, ctr, gctr, queue, wait, wq, wlock, pending,
                   reqs, rd, cs, dead, refs, delivered, lost, rel>>

(* The socket fails. Nothing is marked until the reader sees the failure. *)
SocketFail ==
    /\ SOCKET_FAIL /\ cs = "open"
    /\ cs' = "closed"
    /\ UNCHANGED <<st, failed, sent, ctr, gctr, queue, wait, wq, wlock, pending,
                   reqs, wire, notified, rd, dead, refs, delivered, lost, rel>>

(* The client closes the connection. "demux" marks it closed first. *)
ClientClose ==
    /\ CLIENT_CLOSE /\ cs = "open"
    /\ cs' = "closed"
    /\ dead' = (dead \/ IsDemux)
    /\ UNCHANGED <<st, failed, sent, ctr, gctr, queue, wait, wq, wlock, pending,
                   reqs, wire, notified, rd, refs, delivered, lost, rel>>

-----------------------------------------------------------------------------
(* Readers *)

\* A reader runs while its stream is open ("asis") or from the dial on.
ReaderActive(r) ==
    IF DESIGN = "asis" THEN st[r] = "open" /\ ~failed[r] ELSE TRUE

(* The reader enters ReadMessage. *)
ReadBegin(r) ==
    /\ rd[r] = "idle" /\ ReaderActive(r)
    /\ rd' = [rd EXCEPT ![r] = "reading"]
    /\ UNCHANGED <<st, failed, sent, ctr, gctr, queue, wait, wq, wlock, pending,
                   reqs, wire, notified, cs, dead, refs, delivered, lost, rel>>

\* The streams a response with this id is routed to by reader r.
Owners(r, id) ==
    IF DESIGN = "asis"
    THEN {s \in Streams : s = r /\ <<id, s>> \in pending}
    ELSE {s \in Streams : <<id, s>> \in pending}

(* ReadMessage returns. On a closed socket the read fails: the reader stops
   and fails the waiters it serves ("asis": its own stream, which also
   records the error; otherwise every registration, marking the connection
   closed in the same step). Otherwise the reader routes the message: a
   notification is reported, a response goes to the registration of its id,
   and a response nobody waits for is reported as an orphan. *)
ReadEnd(r) ==
    /\ rd[r] = "reading"
    /\ cs # "open" \/ wire # <<>>
    /\ IF cs # "open"
       THEN LET regs == IF DESIGN = "asis" THEN Own(r) ELSE pending
            IN /\ rd' = [rd EXCEPT ![r] = "done"]
               /\ delivered' = delivered \cup FailAll(regs)
               /\ pending' = pending \ regs
               /\ failed' = IF DESIGN = "asis"
                            THEN [failed EXCEPT ![r] = TRUE]
                            ELSE failed
               /\ dead' = (DESIGN # "asis")
               /\ UNCHANGED <<wire, lost>>
       ELSE LET m == Head(wire)
            IN /\ wire' = Tail(wire)
               /\ rd' = [rd EXCEPT ![r] = "idle"]
               /\ UNCHANGED <<failed, dead>>
               /\ IF m.id = NoId
                  THEN UNCHANGED <<pending, delivered, lost>>
                  ELSE IF Owners(r, m.id) # {}
                  THEN \E s \in Owners(r, m.id) :
                          /\ pending' = pending \ {<<m.id, s>>}
                          /\ delivered' = delivered \cup {<<s, m.id, m.from>>}
                          /\ UNCHANGED lost
                  ELSE /\ lost' = IF <<m.id, m.from>> \in pending
                                  THEN lost \cup {m} ELSE lost
                       /\ UNCHANGED <<pending, delivered>>
    /\ UNCHANGED <<st, sent, ctr, gctr, queue, wait, wq, wlock, reqs, notified,
                   cs, refs, rel>>

-----------------------------------------------------------------------------
StreamNext(s) ==
    \/ Open(s) \/ Register(s) \/ WriteBegin(s) \/ WriteEnd(s)
    \/ RecvBegin(s) \/ RecvResult(s) \/ RecvDone(s) \/ RecvTimeout(s)
    \/ RecvCancel(s) \/ Close(s)

Next ==
    \/ \E s \in Streams : StreamNext(s)
    \/ \E r \in reqs : Respond(r)
    \/ Notify \/ SocketFail \/ ClientClose \/ ReleaseClose
    \/ \E r \in Readers : ReadBegin(r) \/ ReadEnd(r)

Spec == Init /\ [][Next]_vars

(* The server, the readers, the writers and a waiting Recv make progress.
   Timeouts, cancellations, closes and failures are the environment's
   choice and are not forced. *)
FairSpec ==
    /\ Spec
    /\ \A s \in Streams :
          /\ WF_vars(WriteBegin(s)) /\ WF_vars(WriteEnd(s))
          /\ WF_vars(RecvResult(s)) /\ WF_vars(RecvDone(s))
    /\ \A r \in Ids \X Streams : WF_vars(Respond(r))
    /\ \A r \in Readers : WF_vars(ReadBegin(r)) /\ WF_vars(ReadEnd(r))
    /\ WF_vars(ReleaseClose)

-----------------------------------------------------------------------------
(* Properties *)

(* gorilla/websocket allows one concurrent reader per connection. *)
OneReader == Cardinality({r \in Readers : rd[r] = "reading"}) <= 1

(* gorilla/websocket allows one concurrent writer per connection. *)
OneWriter == wlock \in Streams \cup {None}

(* A stream only ever gets the responses to its own requests. Every result
   Recv returns was held in delivered first. *)
OwnResponses ==
    \A d \in delivered : d[3] \in Streams => d[3] = d[1]

(* No response is dropped while its own stream still waits for it. *)
NoLostResponse == lost = {}

(* One registration per id on the connection. *)
UniqueIds == \A p, q \in pending : p[1] = q[1] => p = q

\* The response to request w of stream s can still arrive, or the reader
\* that serves s will fail it.
Coming(s, w) ==
    /\ <<w, s>> \in pending
    /\ LET r == IF DESIGN = "asis" THEN s ELSE "conn"
       IN \/ wq[s] = w
          \/ /\ rd[r] # "done"
             /\ \/ cs # "open"
                \/ <<w, s>> \in reqs
                \/ \E i \in DOMAIN wire : wire[i].id = w /\ wire[i].from = s

(* A waiting Recv can always return without its timer: its result is held,
   the stream is done, or the response or the failure is still coming. *)
NoStuckRecv ==
    \A s \in Streams :
        wait[s] # NoId =>
            \/ \E d \in delivered : d[1] = s /\ d[2] = wait[s]
            \/ st[s] = "closed" \/ failed[s]
            \/ Coming(s, wait[s])

(* Once the connection reader stopped, no registration is left behind,
   except one whose write is still in progress: that write fails. *)
NoLeakAfterStop ==
    (DESIGN # "asis" /\ rd["conn"] = "done")
        => \A p \in pending : wq[p[2]] = p[1]

(* The connection closes once every opened stream is done with it ("demux";
   "demux_split" once its pending close has run). *)
ClosedWhenReleased ==
    (IsDemux /\ rel # "pending" /\ \A s \in Streams : st[s] = "closed")
        => cs = "closed"

(* The last release never closes a connection that a stream acquired in the
   meantime. *)
NoCloseUnderAcquire == rel # "bad"

(* Every waiting Recv eventually returns. *)
RecvReturns == \A s \in Streams : [](wait[s] # NoId => <>(wait[s] = NoId))

=============================================================================
