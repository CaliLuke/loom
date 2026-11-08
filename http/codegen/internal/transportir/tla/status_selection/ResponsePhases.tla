------------------------- MODULE ResponsePhases -------------------------
\* #976 informational, upgrade and final phases on one HTTP exchange with a
\* concurrent informational sender. One writer owner serializes interim
\* publication against every final start (unary result, error, SSE open,
\* WebSocket upgrade, redirect). The final start closes informational admission before
\* any final header is prepared. net/http's automatic 100 Continue is a
\* separate, protocol-owned interim response with its own client budget.
EXTENDS Naturals, FiniteSets, Sequences
CONSTANTS Policy, Limit, MaxAttempts
VARIABLES infoKeys, proto, expect, observed, foreign, kind,
          owner, sink, sendStage, sendCode, snap, hdr, finalStage, auto,
          attempts, reported, wire, idx, authoredSeen, continueSeen, delivered, outcome
vars == <<infoKeys, proto, expect, observed, foreign, kind,
          owner, sink, sendStage, sendCode, snap, hdr, finalStage, auto,
          attempts, reported, wire, idx, authoredSeen, continueSeen, delivered, outcome>>

Exact(c) == [kind |-> "exact", code |-> c]
Class1 == [kind |-> "class", code |-> 0]
NoKey == [kind |-> "none", code |-> 0]
Attempts == {100, 101, 102, 103}
Interim(c) == c >= 100 /\ c <= 199 /\ c # 101
InfoHeaders == {"link"}
\* Final starts and what each publishes.
FinalStatus == [final |-> 200, error |-> 500, sse |-> 200, upgrade |-> 101, redirect |-> 302]
FinalHeaders == [final |-> {"etag"}, error |-> {"problem"}, sse |-> {"event-stream"}, upgrade |-> {"upgrade"}, redirect |-> {"location"}]

SelectInfo(c) == IF ~Interim(c) THEN NoKey
                 ELSE IF Exact(c) \in infoKeys THEN Exact(c)
                 ELSE IF Class1 \in infoKeys THEN Class1 ELSE NoKey
\* Authored emission domain: declared informational keys, never 100 (owned by
\* Expect: 100-continue handling) or 101 (upgrade).
InfoDomain == {c \in Attempts : c \notin {100, 101} /\
                (Exact(c) \in infoKeys \/ (Class1 \in infoKeys /\ Exact(c) \notin infoKeys))}
SentCount == Cardinality({i \in 1..Len(reported) : reported[i] = "sent"})
Authored1xx(w) == Cardinality({i \in 1..Len(w) : Interim(w[i].status) /\ w[i].origin = "authored"})
Locked == Policy # "unlocked" /\ ~(Policy = "redirect-unlocked" /\ kind = "redirect")

Init == /\ infoKeys \in SUBSET {Exact(103), Class1}
        /\ proto \in {"1.0", "1.1", "2"}
        /\ expect \in BOOLEAN
        /\ observed \in BOOLEAN
        /\ foreign \in BOOLEAN
        /\ kind \in {"final", "error", "sse", "upgrade", "redirect"}
        /\ owner = "none" /\ sink = "open" /\ sendStage = "idle" /\ sendCode = 0
        /\ snap = {} /\ hdr = {} /\ finalStage = "none" /\ auto = FALSE
        /\ attempts = 0 /\ reported = <<>> /\ wire = <<>>
        /\ idx = 1 /\ authoredSeen = 0 /\ continueSeen = 0 /\ delivered = <<>> /\ outcome = "none"

Admits(c) ==
  IF foreign THEN TRUE
  ELSE /\ sink = "open"
       /\ (Policy = "http10" \/ proto # "1.0")
       /\ (Policy = "unbounded" \/ SentCount < Limit)
       /\ c \in InfoDomain
ServerVars == <<infoKeys, proto, expect, observed, foreign, kind>>
ClientVars == <<idx, authoredSeen, continueSeen, delivered, outcome>>

\* A retained service goroutine calls the typed emitter at any time.
SendBegin == /\ sendStage = "idle" /\ attempts < MaxAttempts /\ finalStage # "client"
             /\ (Locked => owner = "none")
             /\ \E c \in Attempts :
                  /\ attempts' = attempts + 1
                  /\ IF Admits(c)
                     THEN /\ owner' = IF Locked THEN "send" ELSE owner
                          /\ sendStage' = "snapshot" /\ sendCode' = c
                          /\ snap' = hdr
                          /\ hdr' = InfoHeaders
                          /\ UNCHANGED reported
                     ELSE /\ reported' = Append(reported, "refused")
                          /\ UNCHANGED <<owner, sendStage, sendCode, snap, hdr>>
             /\ UNCHANGED <<ServerVars, sink, finalStage, auto, wire, ClientVars>>
\* net/http writes the live header map; after the final header it drops the call.
SendWrite == /\ sendStage = "snapshot"
             /\ wire' = IF finalStage = "committed" THEN wire
                        ELSE Append(wire, [status |-> sendCode, headers |-> hdr, origin |-> "authored"])
             /\ sendStage' = "written"
             /\ UNCHANGED <<ServerVars, owner, sink, sendCode, snap, hdr, finalStage, auto,
                            attempts, reported, ClientVars>>
SendRestore == /\ sendStage = "written"
               /\ hdr' = IF Policy = "leak" THEN snap \cup InfoHeaders ELSE snap
               /\ sendStage' = "idle"
               /\ owner' = IF Locked THEN "none" ELSE owner
               /\ reported' = Append(reported, "sent")
               /\ UNCHANGED <<ServerVars, sink, sendCode, snap, finalStage, auto, attempts, wire, ClientVars>>

\* Request body read: net/http answers Expect: 100-continue on its own
\* connection writer, without the handler header map.
AutoContinue == /\ expect /\ ~auto /\ finalStage \in {"none", "started", "prepared"} /\ proto # "1.0"
                /\ auto' = TRUE
                /\ wire' = Append(wire, [status |-> 100, headers |-> {}, origin |-> "protocol"])
                /\ UNCHANGED <<ServerVars, owner, sink, sendStage, sendCode, snap, hdr, finalStage,
                               attempts, reported, ClientVars>>

\* Final start: acquire the writer owner and close admission before any final
\* header is touched. The unlocked candidate has no single owner and closes only at WriteHeader.
FinalStart == /\ finalStage = "none" /\ (Locked => owner = "none")
              /\ owner' = IF Locked THEN "final" ELSE owner
              /\ sink' = IF Policy \in {"unlocked", "late"} THEN sink ELSE "closed"
              /\ finalStage' = "started"
              /\ UNCHANGED <<ServerVars, sendStage, sendCode, snap, hdr, auto, attempts, reported, wire, ClientVars>>
FinalPrep == /\ finalStage = "started"
             /\ hdr' = hdr \cup FinalHeaders[kind]
             /\ finalStage' = "prepared"
             /\ UNCHANGED <<ServerVars, owner, sink, sendStage, sendCode, snap, auto, attempts, reported, wire, ClientVars>>
FinalCommit == /\ finalStage = "prepared"
               /\ wire' = IF Policy = "info-final" /\ kind = "final"
                          THEN wire \o <<[status |-> 103, headers |-> hdr, origin |-> "authored"],
                                         [status |-> 200, headers |-> hdr \ FinalHeaders[kind], origin |-> "final"]>>
                          ELSE Append(wire, [status |-> FinalStatus[kind], headers |-> hdr, origin |-> "final"])
               /\ reported' = IF Policy = "info-final" /\ kind = "final" THEN Append(reported, "sent") ELSE reported
               /\ sink' = IF Policy = "late" THEN sink ELSE "closed"
               /\ finalStage' = "committed"
               /\ owner' = IF Locked THEN "none" ELSE owner
               /\ UNCHANGED <<ServerVars, sendStage, sendCode, snap, hdr, auto, attempts, ClientVars>>

Receive == /\ finalStage = "committed" /\ sendStage = "idle"
           /\ finalStage' = "client"
           /\ UNCHANGED <<ServerVars, owner, sink, sendStage, sendCode, snap, hdr, auto,
                          attempts, reported, wire, ClientVars>>

\* Client: authored interim responses are budgeted by Limit; the automatic
\* 100 Continue has its own budget of one (shared-budget counts it in Limit).
ExpectedOutcome == "final-" \o kind
ClientStep ==
  /\ finalStage = "client" /\ idx <= Len(wire) /\ outcome = "none"
  /\ LET e == wire[idx] s == e.status IN
     IF Interim(s)
     THEN IF ~observed
          THEN /\ idx' = idx + 1
               /\ UNCHANGED <<authoredSeen, continueSeen, delivered, outcome>>
          ELSE IF s = 100 /\ Policy # "shared-budget"
          THEN IF continueSeen >= 1
               THEN /\ outcome' = "too-many" /\ UNCHANGED <<idx, authoredSeen, continueSeen, delivered>>
               ELSE /\ continueSeen' = 1 /\ idx' = idx + 1
                    /\ UNCHANGED <<authoredSeen, delivered, outcome>>
          ELSE IF Policy # "unbounded" /\ authoredSeen >= Limit
          THEN /\ outcome' = "too-many" /\ UNCHANGED <<idx, authoredSeen, continueSeen, delivered>>
          ELSE /\ authoredSeen' = authoredSeen + 1
               /\ delivered' = IF SelectInfo(s) # NoKey
                               THEN Append(delivered, [key |-> SelectInfo(s), status |-> s])
                               ELSE delivered
               /\ idx' = idx + 1
               /\ UNCHANGED <<continueSeen, outcome>>
     ELSE /\ outcome' = IF s = FinalStatus[kind] /\ e.headers = FinalHeaders[kind] THEN ExpectedOutcome
                        ELSE "mismatch"
          /\ UNCHANGED <<idx, authoredSeen, continueSeen, delivered>>
  /\ UNCHANGED <<ServerVars, owner, sink, sendStage, sendCode, snap, hdr, finalStage, auto,
                 attempts, reported, wire>>

Next == \/ SendBegin \/ SendWrite \/ SendRestore \/ AutoContinue
        \/ FinalStart \/ FinalPrep \/ FinalCommit \/ Receive \/ ClientStep
        \/ (outcome # "none" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars

FinalIndex == CHOOSE i \in 1..Len(wire) : wire[i].origin = "final"
HasFinal == \E i \in 1..Len(wire) : wire[i].origin = "final"
Done == outcome # "none"

\* Interim responses precede the single final response and are never results.
InterimBeforeFinal == ~foreign /\ HasFinal => \A i \in 1..Len(wire) : i <= FinalIndex
InterimNeverTerminal == outcome \in {"none", "final-final", "final-error", "final-sse", "final-upgrade", "final-redirect", "mismatch", "too-many"}
\* Concurrent interim publication never changes the final contract of a Loom peer.
FinalAgreement == Done /\ ~foreign => outcome = ExpectedOutcome
FinalHeadersIsolated == ~foreign /\ HasFinal => wire[FinalIndex].headers = FinalHeaders[kind]
No1xxToHTTP10 == ~foreign /\ proto = "1.0" => Authored1xx(wire) = 0
NoAuthored100or101 == ~foreign => \A i \in 1..Len(wire) : wire[i].origin = "authored" => wire[i].status \notin {100, 101}
\* Every emission the service was told succeeded reached the wire before final.
EmitHonest == ~foreign /\ sendStage = "idle" => SentCount = Authored1xx(wire)
\* Bounded client acceptance; Loom peers never trip the bound.
ClientBounded == authoredSeen <= Limit /\ continueSeen <= 1
LoomPeersNeverAbort == ~foreign => outcome # "too-many"
DeliveredSelected == \A i \in 1..Len(delivered) :
                       /\ Interim(delivered[i].status)
                       /\ delivered[i].key = SelectInfo(delivered[i].status)
=============================================================================
