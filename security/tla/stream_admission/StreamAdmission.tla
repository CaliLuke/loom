----------------------------- MODULE StreamAdmission -----------------------------
EXTENDS Naturals, Sequences, TLC
CONSTANT Mode
VARIABLES allMetadata, good, hasMiddleware, frameValid, phase, frame,
          principal, authenticated, originalLive, callbackLive, initialCalls,
          complete, accessed, intercepted, invoked, mutated
vars == <<allMetadata, good, hasMiddleware, frameValid, phase, frame,
          principal, authenticated, originalLive, callbackLive, initialCalls,
          complete, accessed, intercepted, invoked, mutated>>
Winner == IF good[1] THEN 1 ELSE IF good[2] THEN 2 ELSE 0
ReadyWinner == IF allMetadata THEN Winner ELSE IF good[2] THEN 2 ELSE 0
EffectiveLive == IF Mode = "lost-context" THEN callbackLive ELSE originalLive /\ callbackLive
Init == /\ allMetadata \in BOOLEAN
        /\ good \in [1..2 -> BOOLEAN]
        /\ hasMiddleware \in BOOLEAN
        /\ frameValid \in BOOLEAN
        /\ phase = IF Mode = "legacy" \/ (~allMetadata /\ Mode # "eager-available") THEN "wait" ELSE "auth"
        /\ frame = FALSE /\ complete = FALSE /\ accessed = FALSE
        /\ intercepted = FALSE /\ invoked = FALSE /\ mutated = FALSE
        /\ principal = 0 /\ authenticated = FALSE
        /\ originalLive = TRUE /\ callbackLive = TRUE /\ initialCalls = 0
Auth == /\ phase = "auth"
        /\ LET chosen == IF Mode = "eager-available" /\ ~frame THEN ReadyWinner ELSE Winner
           IN /\ principal' = chosen
              /\ authenticated' = (chosen # 0)
              /\ phase' = IF ~EffectiveLive THEN "done" ELSE
                    IF chosen # 0 THEN (IF complete THEN "access" ELSE IF frame THEN "validate" ELSE "wait")
                    ELSE IF ~allMetadata /\ ~frame THEN "wait" ELSE "done"
        /\ initialCalls' = initialCalls + 1
        /\ UNCHANGED <<allMetadata, good, hasMiddleware, frameValid, frame,
              originalLive, callbackLive, complete, accessed, intercepted, invoked, mutated>>
Frame == /\ phase = "wait" /\ EffectiveLive
         /\ frame' = TRUE
         /\ phase' = "validate"
         /\ UNCHANGED <<allMetadata, good, hasMiddleware, frameValid, principal,
              authenticated, originalLive, callbackLive, initialCalls,
              complete, accessed, intercepted, invoked, mutated>>
Validate == /\ phase = "validate"
            /\ complete' = frameValid
            /\ phase' = IF ~EffectiveLive \/ ~frameValid THEN "done" ELSE
                  IF authenticated /\ Mode # "admission" THEN "access" ELSE "auth"
            /\ UNCHANGED <<allMetadata, good, hasMiddleware, frameValid, frame, principal,
                 authenticated, originalLive, callbackLive, initialCalls,
                 accessed, intercepted, invoked, mutated>>
Access == /\ phase = "access"
          /\ accessed' = EffectiveLive
          /\ phase' = IF ~EffectiveLive THEN "done" ELSE IF hasMiddleware THEN "middleware" ELSE "invoke"
          /\ UNCHANGED <<allMetadata, good, hasMiddleware, frameValid, frame, principal,
               authenticated, originalLive, callbackLive, initialCalls, complete,
               intercepted, invoked, mutated>>
Middleware == /\ phase = "middleware" /\ EffectiveLive
              /\ intercepted' = TRUE
              /\ mutated' \in BOOLEAN
              /\ phase' = IF Mode = "no-recheck" THEN "invoke" ELSE "recheck"
              /\ UNCHANGED <<allMetadata, good, hasMiddleware, frameValid, frame, principal,
                   authenticated, originalLive, callbackLive, initialCalls, complete,
                   accessed, invoked>>
Recheck == /\ phase = "recheck"
           /\ phase' = IF EffectiveLive /\ ~mutated THEN "invoke" ELSE "done"
           /\ UNCHANGED <<allMetadata, good, hasMiddleware, frameValid, frame, principal,
                authenticated, originalLive, callbackLive, initialCalls, complete,
                accessed, intercepted, invoked, mutated>>
Invoke == /\ phase = "invoke" /\ EffectiveLive
          /\ invoked' = TRUE /\ phase' = "done"
          /\ UNCHANGED <<allMetadata, good, hasMiddleware, frameValid, frame, principal,
               authenticated, originalLive, callbackLive, initialCalls, complete,
               accessed, intercepted, mutated>>
Cancel == /\ originalLive /\ phase # "done"
          /\ originalLive' = FALSE
          /\ UNCHANGED <<allMetadata, good, hasMiddleware, frameValid, phase, frame, principal,
               authenticated, callbackLive, initialCalls, complete,
               accessed, intercepted, invoked, mutated>>
Stop == /\ ~EffectiveLive /\ phase # "done"
        /\ phase' = "done"
        /\ UNCHANGED <<allMetadata, good, hasMiddleware, frameValid, frame, principal,
             authenticated, originalLive, callbackLive, initialCalls, complete,
             accessed, intercepted, invoked, mutated>>
Next == Auth \/ Frame \/ Validate \/ Access \/ Middleware \/ Recheck \/ Invoke \/ Cancel \/ Stop
Spec == Init /\ [][Next]_vars
PromptRejection == allMetadata /\ phase = "wait" => authenticated
DeclaredWinner == authenticated => principal = Winner
NoInitialRepeat == initialCalls <= 1
CompleteConsumers == (accessed \/ intercepted \/ invoked) => complete /\ authenticated
CancelledInvocation == invoked => originalLive
MiddlewareRechecked == invoked => ~mutated
=============================================================================
