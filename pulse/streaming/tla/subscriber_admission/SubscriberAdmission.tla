------------------------ MODULE SubscriberAdmission ------------------------
EXTENDS Naturals, FiniteSets
CONSTANT SealedAdmission
VARIABLES phase, requested, registered, closedChannels, started, lateWorker, lateReturned
vars == <<phase, requested, registered, closedChannels, started, lateWorker, lateReturned>>
Subscribers == {1, 2}
Init == /\ phase = "open" /\ requested = {} /\ registered = {}
        /\ closedChannels = {} /\ started = FALSE /\ lateWorker = FALSE /\ lateReturned = {}
Subscribe(s) == /\ s \notin requested
                /\ requested' = requested \cup {s}
                /\ lateReturned' = IF phase # "open" THEN lateReturned \cup {s} ELSE lateReturned
                /\ IF SealedAdmission /\ phase # "open"
                      THEN /\ closedChannels' = closedChannels \cup {s}
                           /\ UNCHANGED <<registered, started, lateWorker>>
                      ELSE /\ registered' = registered \cup {s}
                           /\ started' = TRUE
                           /\ lateWorker' = (lateWorker \/ (~started /\ phase # "open"))
                           /\ UNCHANGED closedChannels
                /\ UNCHANGED phase
Stop == /\ phase = "open" /\ phase' = "closing"
        /\ UNCHANGED <<requested, registered, closedChannels, started, lateWorker, lateReturned>>
Drain == /\ phase = "closing" /\ phase' = "closed"
         /\ closedChannels' = closedChannels \cup registered
         /\ registered' = {}
         /\ UNCHANGED <<requested, started, lateWorker, lateReturned>>
Unsubscribe(s) == /\ s \in registered
                  /\ registered' = registered \ {s}
                  /\ closedChannels' = closedChannels \cup {s}
                  /\ UNCHANGED <<phase, requested, started, lateWorker, lateReturned>>
Next == Stop \/ Drain \/ (\E s \in Subscribers: Subscribe(s) \/ Unsubscribe(s))
Spec == Init /\ [][Next]_vars /\ WF_vars(Drain)
TypeOK == /\ phase \in {"open", "closing", "closed"}
          /\ requested \subseteq Subscribers /\ registered \subseteq requested
          /\ closedChannels \subseteq requested /\ lateReturned \subseteq requested /\ started \in BOOLEAN /\ lateWorker \in BOOLEAN
NoLateWorker == ~lateWorker
LateChannelsClosed == lateReturned \subseteq closedChannels
ClosedOwnsNoSubscriber == phase = "closed" => registered = {}
ClosedChannels == phase = "closed" => requested = closedChannels
ClosingFinishes == phase = "closing" ~> phase = "closed"
=============================================================================
