----------------------------- MODULE InitialSend -----------------------------
EXTENDS TLC
CONSTANT PreserveEOF
VARIABLES kind, hasPayload, sendResult, finalStatus, phase, outcome
vars == <<kind, hasPayload, sendResult, finalStatus, phase, outcome>>

Init ==
  /\ kind \in {"client", "bidirectional"}
  /\ hasPayload \in BOOLEAN
  /\ sendResult \in {"ok", "eof", "local-error"}
  /\ finalStatus \in {"ok", "denied", "canceled", "empty"}
  /\ phase = "open"
  /\ outcome = "pending"

Open ==
  /\ phase = "open"
  /\ phase' = IF hasPayload THEN "send" ELSE "returned"
  /\ UNCHANGED <<kind, hasPayload, sendResult, finalStatus, outcome>>

Send ==
  /\ phase = "send"
  /\ IF sendResult = "ok" \/ (PreserveEOF /\ sendResult = "eof")
        THEN /\ phase' = "returned"
             /\ outcome' = "pending"
        ELSE /\ phase' = "dropped"
             /\ outcome' = sendResult
  /\ UNCHANGED <<kind, hasPayload, sendResult, finalStatus>>

Receive ==
  /\ phase = "returned"
  /\ phase' = "done"
  /\ outcome' = finalStatus
  /\ UNCHANGED <<kind, hasPayload, sendResult, finalStatus>>

Next == Open \/ Send \/ Receive
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)

TypeOK ==
  /\ kind \in {"client", "bidirectional"}
  /\ hasPayload \in BOOLEAN
  /\ sendResult \in {"ok", "eof", "local-error"}
  /\ finalStatus \in {"ok", "denied", "canceled", "empty"}
  /\ phase \in {"open", "send", "returned", "dropped", "done"}
  /\ outcome \in {"pending", "eof", "local-error", "ok", "denied", "canceled", "empty"}
FinalStatusRecoverable == ~(phase = "dropped" /\ sendResult = "eof")
LocalFailuresVisible ==
  (hasPayload /\ sendResult = "local-error" /\ phase \in {"dropped", "done"})
    => (phase = "dropped" /\ outcome = "local-error")
ReceivedStatus == phase = "done" => outcome = finalStatus
EventuallyTerminal == <>(phase \in {"done", "dropped"})
=============================================================================
