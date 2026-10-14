------------------------- MODULE SinkActivation -------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Activation
VARIABLES phase, requested, registered, closedChannels, activated,
          workers, starts, observed, early
vars == <<phase, requested, registered, closedChannels, activated,
          workers, starts, observed, early>>
Subscribers == {1, 2}
Producers == {"read", "claim"}
InitialProducers == CASE Activation = "eager" -> Producers
                     [] Activation = "read-only" -> {"claim"}
                     [] OTHER -> {}
Init == /\ phase = "open" /\ requested = {} /\ registered = {}
        /\ closedChannels = {} /\ activated = FALSE
        /\ workers = InitialProducers \cup {"keepalive"}
        /\ starts = [p \in Producers |-> IF p \in InitialProducers THEN 1 ELSE 0]
        /\ observed = {} /\ early = {}
Subscribe(s) ==
 /\ s \notin requested
 /\ requested' = requested \cup {s}
 /\ IF phase # "open"
       THEN /\ closedChannels' = closedChannels \cup {s}
            /\ UNCHANGED <<registered, activated, workers, starts>>
       ELSE /\ registered' = registered \cup {s}
            /\ activated' = TRUE
            /\ IF activated
                  THEN UNCHANGED <<workers, starts>>
                  ELSE /\ workers' = workers \cup Producers
                       /\ starts' = [p \in Producers |-> IF p \in InitialProducers THEN starts[p] ELSE starts[p] + 1]
            /\ UNCHANGED closedChannels
 /\ UNCHANGED <<phase, observed, early>>
Advance(p) == /\ phase = "open" /\ p \in workers /\ p \notin observed
              /\ observed' = observed \cup {p}
              /\ early' = IF activated THEN early ELSE early \cup {p}
              /\ UNCHANGED <<phase, requested, registered, closedChannels, activated, workers, starts>>
Unsubscribe(s) == /\ s \in registered
                  /\ registered' = registered \ {s}
                  /\ closedChannels' = closedChannels \cup {s}
                  /\ UNCHANGED <<phase, requested, activated, workers, starts, observed, early>>
Stop == /\ phase = "open" /\ phase' = "closing"
        /\ UNCHANGED <<requested, registered, closedChannels, activated, workers, starts, observed, early>>
Exit(w) == /\ phase = "closing" /\ w \in workers
           /\ workers' = workers \ {w}
           /\ UNCHANGED <<phase, requested, registered, closedChannels, activated, starts, observed, early>>
Finish == /\ phase = "closing" /\ workers = {} /\ phase' = "closed"
          /\ closedChannels' = closedChannels \cup registered /\ registered' = {}
          /\ UNCHANGED <<requested, activated, workers, starts, observed, early>>
Next == Stop \/ Finish \/ (\E s \in Subscribers: Subscribe(s) \/ Unsubscribe(s))
        \/ (\E p \in Producers: Advance(p)) \/ (\E w \in Producers \cup {"keepalive"}: Exit(w))
Spec == Init /\ [][Next]_vars /\ WF_vars(Finish)
        /\ (\A w \in Producers \cup {"keepalive"}: WF_vars(Exit(w)))
TypeOK == /\ phase \in {"open", "closing", "closed"}
          /\ requested \subseteq Subscribers /\ registered \subseteq requested
          /\ closedChannels \subseteq requested /\ activated \in BOOLEAN
          /\ workers \subseteq Producers \cup {"keepalive"}
          /\ starts \in [Producers -> 0..1]
          /\ observed \subseteq Producers /\ early \subseteq observed
NoEarlyAdvance == early = {}
LateChannelsClosed == phase = "closed" => requested = closedChannels
ClosedWorkersEmpty == phase = "closed" => workers = {}
ClosingFinishes == phase = "closing" ~> phase = "closed"
=============================================================================
