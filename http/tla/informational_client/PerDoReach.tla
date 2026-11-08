---------------------------- MODULE PerDoReach ----------------------------
EXTENDS PerDo
NoRetainedErrorAcrossReset == ~(
  s.returned /\ s.snapshot = "observerA" /\ s.result[1] = "observerA" /\
  (\E i \in 1..Len(s.history): s.history[i].exchange = 1 /\ s.history[i].err = "observerA") /\
  (\E i \in 1..Len(s.history): s.history[i].exchange = 2 /\ s.history[i].observer = "observerB" /\
      ~s.history[i].admitted /\ ~s.history[i].observerCalled /\ s.history[i].err = "observerA"))
NoHealthyBudgetReset == ~(
  s.read /\ s.readOK /\ s.caller = "ok" /\ s.callerCalls = 4 /\
  Len(s.failures) = 0 /\
  (\A x \in 1..2: Count(s.history,x,TRUE) = 1 /\ Count(s.history,x,FALSE) = 1))
NoThreeErrorJoin == ~(s.returned /\ s.errorBodyClosed /\
  s.result = <<"observerA","caller","close">>)
NoLateCaller == ~(s.returned /\ s.lateCaller)
NoPostFinishHook == ~(s.returned /\ s.lateHook /\ s.result[1] = "observerA" /\
  ~s.history[Len(s.history)].admitted /\ ~s.history[Len(s.history)].observerCalled)
NoRacedSuccessClose == ~(s.returned /\ s.snapshot # "" /\ s.doErr = "" /\ s.errorBodyClosed)
NoCleanResetFailure == ~(
  s.returned /\ s.snapshot = "observerA" /\ s.cleanReset /\
  (\E i \in 1..Len(s.history): s.history[i].exchange = 1) /\
  (\A i \in 1..Len(s.history): s.history[i].exchange = 1 => s.history[i].err = "") /\
  (\E i \in 1..Len(s.history): s.history[i].exchange = 2 /\ s.history[i].err = "observerA") /\
  s.completedHooks = Len(s.history))
NoHundredObserverFailure == ~(s.returned /\ s.snapshot = "observerA" /\
  (\E i \in 1..Len(s.history): s.history[i].code = 100 /\ s.history[i].err = "observerA"))
NoTwoOtherAccepted == ~(s.returnedBody /\ Count(s.history,1,FALSE) = 2 /\ Len(s.failures) = 0)
NoThirdOtherOverflow == ~(s.returned /\ s.snapshot = "limit" /\ Count(s.history,1,FALSE) = 3)
NoIndependentFullBudgets == ~(s.returnedBody /\ Count(s.history,1,TRUE) = 1 /\
  Count(s.history,1,FALSE) = 2 /\ Len(s.failures) = 0)
NoPostFinishCaller == ~(s.returned /\ s.lateHook /\ s.pc = "ready" /\
  s.caller = "ok" /\ s.callerCalls = Len(s.history) /\ s.lateCaller)
NoLateWriteAtLimit == ~(s.lateWrite /\ s.returned /\ s.snapshot = "limit" /\
  Count(s.history,1,FALSE) = Limit + 1)
NoLateWriteHundredLimit == ~(s.lateWrite /\ s.returned /\ s.snapshot = "limit" /\
  Count(s.history,1,TRUE) = 2)
NoLateWritePreserved == ~(s.lateWrite /\ s.firstByte /\ s.pc = "ready" /\
  s.stored = "" /\ s.other = Limit /\ Count(s.history,s.exchange,FALSE) = Limit)
=============================================================================
