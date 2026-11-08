------------------------------- MODULE PerDo -------------------------------
EXTENDS Naturals, Sequences, FiniteSets, TLC
CONSTANTS Mode, Exchanges, MaxEvents, Limit, ObserverErrors
VARIABLE s
vars == <<s>>
Codes == {100, 102, 103}
Errors == {"", "observerA", "observerB", "limit", "caller", "cancel", "close"}
Count(h, x, hundred) == Cardinality({i \in 1..Len(h):
    h[i].exchange = x /\ h[i].admitted /\ ((h[i].code = 100) = hundred)})
Deliveries(h,x) == Cardinality({i \in 1..Len(h): h[i].exchange = x})
Nonempty(q) == SelectSeq(q, LAMBDA e: e # "")
First(q) == IF Len(q) = 0 THEN "" ELSE q[1]
PriorErrors(h) == Nonempty([i \in 1..Len(h) |-> IF h[i].admitted THEN h[i].err ELSE ""])
ExpectedError(h, x, code, observer) ==
    IF Len(PriorErrors(h)) > 0 THEN First(PriorErrors(h))
    ELSE IF (code = 100 /\ Count(h,x,TRUE) >= 1)
       \/ (code # 100 /\ Count(h,x,FALSE) >= Limit)
    THEN "limit"
    ELSE IF code \in {100,103} THEN observer ELSE ""

Init == \E caller \in {"absent", "ok", "fail"}, body \in {"stream", "none"},
           closeErr \in {"", "close"}:
  s = [pc |-> "ready", doPC |-> "running", caller |-> caller, bodyKind |-> body,
       closeErr |-> closeErr, errorBody |-> FALSE,
       exchange |-> 1, hundred |-> 0, other |-> 0,
       firstByte |-> FALSE, wroteHeaders |-> FALSE, lateWrite |-> FALSE,
       history |-> <<>>, failures |-> <<>>, stored |-> "",
       pending |-> "", canceled |-> FALSE, abortCanceled |-> FALSE,
       callerCalls |-> 0, completedHooks |-> 0, transportErr |-> "",
       doErr |-> "", snapshot |-> "", seenFailures |-> <<>>, result |-> <<>>,
       returned |-> FALSE, returnedBody |-> FALSE, errorBodyClosed |-> FALSE,
       eof |-> FALSE, closeStage |-> 0, closeResult |-> "",
       bodyOnce |-> FALSE, bodyCancels |-> 0,
       read |-> FALSE, readOK |-> FALSE, lateCaller |-> FALSE, lateHook |-> FALSE, cleanReset |-> FALSE]

\* Physical exchange advance is separate from transport callbacks. Retrying
\* after abort remains a conservative stress schedule, not native retry policy.
NextExchange ==
  /\ s.exchange < Exchanges /\ s.doPC = "running" /\ s.firstByte
  /\ s' = [s EXCEPT !.exchange = @ + 1,
      !.firstByte = FALSE, !.wroteHeaders = FALSE,
      !.cleanReset = @ \/ (Len(s.history) > 0 /\ Len(s.failures) = 0 /\
         s.completedHooks = Len(s.history) /\ ~s.canceled)]

\* Native transport delivers this once per response/stream, before any 1xx.
\* Only counters reset; the first retained error is never an exchange value.
FirstResponseByte ==
  /\ ~s.firstByte /\ s.doPC = "running"
  /\ s' = [s EXCEPT !.firstByte = TRUE,
      !.hundred = IF Mode \in {"no-reset", "late-write-reset"} THEN @ ELSE 0,
      !.other = IF Mode \in {"no-reset", "late-write-reset"} THEN @ ELSE 0,
      !.stored = IF Mode = "reset-error" THEN "" ELSE @]

\* HTTP/2 write/read goroutines may deliver this after one or more 1xx.
\* Candidate ignores it; old reset behavior is a deliberate current mutant.
WroteHeaders ==
  /\ ~s.wroteHeaders
  /\ s' = [s EXCEPT !.wroteHeaders = TRUE,
      !.hundred = IF Mode = "late-write-reset" THEN 0 ELSE @,
      !.other = IF Mode = "late-write-reset" THEN 0 ELSE @,
      !.lateWrite = @ \/ (s.firstByte /\ Deliveries(s.history,s.exchange) > 0)]

\* Incoming hooks serialize. 100 and103 are declared;102 is counted only.
\* Buffered hooks can run after Do/finish; a retained error short-circuits
\* Loom admission/observers. History flags are proof observations, not runtime state.
Hook(code, observer) ==
  /\ s.pc = "ready" /\ s.transportErr = "" /\ s.firstByte
  /\ (s.doPC = "running" \/ s.abortCanceled)
  /\ Deliveries(s.history,s.exchange) < MaxEvents
  /\ LET admitted == s.stored = "" \/ Mode = "late-observer"
         excess == IF Mode = "shared-budget"
                     THEN s.hundred + s.other >= Limit
                     ELSE IF Mode = "swapped-budget"
                       THEN IF code = 100 THEN s.hundred >= Limit ELSE s.other >= 1
                       ELSE IF code = 100 THEN s.hundred >= 1
                            ELSE IF Mode = "hundred-charges-other" THEN s.hundred + s.other >= Limit
                            ELSE s.other >= Limit
         err == IF ~admitted THEN s.stored
                ELSE IF excess THEN "limit" ELSE IF code \in {100,103} THEN observer ELSE ""
     IN s' = [s EXCEPT
       !.pc = "cancel", !.pending = err,
       !.hundred = @ + IF admitted /\ code = 100 THEN 1 ELSE 0,
       !.other = @ + IF admitted /\ code # 100 THEN 1 ELSE 0,
       !.history = Append(@,[exchange |-> s.exchange, code |-> code,
                            observer |-> observer, err |-> err, admitted |-> admitted,
                            observerCalled |-> admitted /\ ~excess /\ code \in {100,103}]),
       !.failures = IF ~admitted \/ err = "" THEN @ ELSE Append(@,err),
       !.stored = IF Mode \in {"plain-return","cancel-before-store"} THEN @
                  ELSE IF @ = "" THEN err ELSE @,
       !.lateHook = @ \/ s.returned]

CancelHook ==
  /\ s.pc = "cancel"
  /\ s' = [s EXCEPT !.pc = IF Mode = "cancel-before-store" THEN "store" ELSE "caller",
       !.canceled = @ \/ (Mode # "plain-return" /\ s.pending # ""),
       !.abortCanceled = @ \/ (Mode # "plain-return" /\ s.pending # "")]

\* Defective ordering only: finish can snapshot before this delayed store.
DelayedStore ==
  /\ s.pc = "store"
  /\ s' = [s EXCEPT !.pc = "caller", !.stored = IF @ = "" THEN s.pending ELSE @]

\* Go compose2to1 returns f2, or f1 when f2 is absent. Never manually replay.
CallerHook ==
  /\ s.pc = "caller"
  /\ LET returned == IF s.caller = "absent" THEN s.pending
                     ELSE IF s.caller = "fail" THEN "caller" ELSE ""
     IN s' = [s EXCEPT !.pc = "ready",
       !.callerCalls = @ + IF s.caller = "absent" THEN 0
                          ELSE IF Mode = "double-hook" THEN 2 ELSE 1,
       !.completedHooks = @ + 1,
       !.transportErr = IF @ = "" THEN returned ELSE @,
       !.lateCaller = @ \/ (s.caller # "absent" /\ s.doPC # "running")]

\* Do and hook progress are independent after Loom's cancellation. A response
\* can win the cancellation race. nil Do error always has a response/body.
DoReturns ==
  /\ s.doPC = "running"
  /\ (s.pc = "ready" \/ s.abortCanceled)
  /\ \E e \in (IF s.abortCanceled THEN {"", "cancel", s.transportErr} ELSE {s.transportErr}),
        hasResponse \in BOOLEAN:
       /\ (e = "" => hasResponse)
       /\ s' = [s EXCEPT !.doPC = "finish", !.doErr = e, !.errorBody = hasResponse]

\* Synchronized owner read occurs in finish, not in Do. Later callbacks never
\* rewrite snapshot/result. A nil Do error does not waive response cleanup.
Finish ==
  /\ s.doPC = "finish"
  /\ LET own == s.stored
         closeBody == own # "" /\ s.errorBody /\ ~(Mode = "skip-success-close" /\ s.doErr = "")
         closeError == IF closeBody /\ Mode # "drop-close-error" THEN s.closeErr ELSE ""
         result == IF own # ""
                   THEN IF Mode = "reverse-join" THEN Nonempty(<<s.doErr,own,closeError>>)
                        ELSE Nonempty(<<own,s.doErr,closeError>>)
                   ELSE Nonempty(<<s.doErr>>)
         success == Len(result) = 0
     IN s' = [s EXCEPT !.doPC = "returned", !.returned = TRUE,
       !.snapshot = own, !.seenFailures = s.failures,
       !.result = result, !.returnedBody = success /\ s.errorBody,
       !.errorBodyClosed = closeBody,
       !.canceled = @ \/ ~success \/ (Mode = "eager-cancel" /\ success)]

Read ==
  /\ s.returnedBody /\ s.bodyKind = "stream"
  /\ ~s.read /\ ~s.eof /\ s.closeStage = 0
  /\ s' = [s EXCEPT !.read = TRUE, !.readOK = ~s.canceled]

\* Atomic test-and-set abstracts a correctly synchronized once primitive.
\* This is not a proof of a racy read/modify/write implementation.
EOF ==
  /\ s.returnedBody /\ s.bodyKind = "stream" /\ ~s.eof
  /\ s' = [s EXCEPT !.eof = TRUE, !.canceled = TRUE,
      !.bodyCancels = @ + IF ~s.bodyOnce \/ Mode = "double-cancel" THEN 1 ELSE 0,
      !.bodyOnce = TRUE]
CloseBegin ==
  /\ s.returnedBody /\ s.closeStage = 0
  /\ s' = [s EXCEPT !.closeStage = 1,
      !.closeResult = IF Mode = "changed-close-error" THEN "" ELSE s.closeErr]
CloseEnd ==
  /\ s.returnedBody /\ s.closeStage = 1
  /\ s' = [s EXCEPT !.closeStage = 2, !.canceled = TRUE,
      !.bodyCancels = @ + IF ~s.bodyOnce \/ Mode = "double-cancel" THEN 1 ELSE 0,
      !.bodyOnce = TRUE]

Next == NextExchange \/ FirstResponseByte \/ WroteHeaders \/ (\E code \in Codes, observer \in ObserverErrors: Hook(code,observer))
        \/ CancelHook \/ DelayedStore \/ CallerHook \/ DoReturns \/ Finish \/ Read \/ EOF \/ CloseBegin \/ CloseEnd
Spec == Init /\ [][Next]_vars
TypeOK ==
  /\ s.pc \in {"ready","cancel","store","caller"}
  /\ s.doPC \in {"running","finish","returned"}
  /\ s.exchange \in 1..Exchanges
  /\ s.hundred \in 0..(Exchanges*MaxEvents) /\ s.other \in 0..(Exchanges*MaxEvents)
  /\ s.stored \in Errors /\ s.pending \in Errors /\ s.doErr \in Errors
  /\ s.bodyCancels \in 0..2 /\ s.closeStage \in 0..2
  /\ Len(s.history) <= Exchanges*MaxEvents /\ s.completedHooks <= Len(s.history)
FirstErrorRetained == s.stored = First(s.failures)
BudgetReset == s.firstByte =>
  s.hundred = Count(s.history,s.exchange,TRUE) /\ s.other = Count(s.history,s.exchange,FALSE)
BudgetsIndependent == \A i \in 1..Len(s.history):
  LET h == s.history[i] IN h.err = ExpectedError(SubSeq(s.history,1,i-1),h.exchange,h.code,h.observer)
NoObserverAfterAbort == \A i \in 1..Len(s.history):
  Len(PriorErrors(SubSeq(s.history,1,i-1))) > 0 => ~s.history[i].observerCalled
NoAdmissionAfterAbort == \A i \in 1..Len(s.history):
  Len(PriorErrors(SubSeq(s.history,1,i-1))) > 0 => ~s.history[i].admitted
CallerExactlyOnce == s.callerCalls = IF s.caller = "absent" THEN 0 ELSE s.completedHooks
AbortPreserved == (s.returned /\ Len(s.seenFailures) > 0) =>
  /\ Len(s.result) > 0 /\ s.result[1] = First(s.seenFailures) /\ ~s.returnedBody
ErrorJoin == (s.returned /\ s.snapshot # "") =>
  /\ s.result = Nonempty(<<s.snapshot,s.doErr,IF s.errorBody THEN s.closeErr ELSE "">>)
  /\ s.errorBodyClosed = s.errorBody
CallerOnlyError == (s.returned /\ Len(s.seenFailures) = 0) => s.result = Nonempty(<<s.doErr>>)
StreamingAlive == (s.returnedBody /\ s.bodyKind = "stream" /\ ~s.eof /\ s.closeStage = 0) => ~s.canceled
ReadSurvives == s.read => s.readOK
BodyCancelOnce == s.bodyCancels <= 1
BodyRelease == (s.returnedBody /\ (s.eof \/ s.closeStage = 2)) => s.bodyOnce /\ s.canceled
CloseIdentity == s.closeStage > 0 => s.closeResult = s.closeErr
ResponsePresence == s.doPC # "running" /\ s.doErr = "" => s.errorBody

NoCrossExchangeFailure == ~(s.returned /\ s.exchange = 2 /\ Len(s.seenFailures) = 1 /\
  (\E i \in 1..Len(s.history): s.history[i].exchange = 2 /\ ~s.history[i].admitted))
NoSuccessfulLateRead == ~(s.read /\ s.readOK /\ s.exchange = 2)
NoEOFCloseRace == ~(s.eof /\ s.closeStage = 2 /\ s.bodyCancels = 1)
NoBodylessRelease == ~(s.returnedBody /\ s.bodyKind = "none" /\ s.closeStage = 2 /\ s.bodyCancels = 1)
=============================================================================
