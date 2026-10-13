-------------------------- MODULE BoundedOperations --------------------------
EXTENDS Naturals, FiniteSets, TLC
CONSTANT Mode
Directions == {"send", "recv"}
VARIABLES native, call, terminal, handlerReturned, attempts, orphaned, observed, terminalSuccess
vars == <<native, call, terminal, handlerReturned, attempts, orphaned, observed, terminalSuccess>>
Init == /\ native = [d \in Directions |-> "idle"]
        /\ call = [d \in Directions |-> "idle"]
        /\ terminal = FALSE /\ handlerReturned = FALSE
        /\ attempts = 0 /\ orphaned = 0
        /\ observed = FALSE /\ terminalSuccess = FALSE
Start(d) == /\ ~terminal /\ call[d] = "idle"
            /\ native' = [native EXCEPT ![d] = "blocked"]
            /\ call' = [call EXCEPT ![d] = "waiting"]
            /\ UNCHANGED <<terminal, handlerReturned, attempts, orphaned, observed, terminalSuccess>>
\* CloseAndRecv reserves both directions atomically. Two occupied slots
\* conservatively represent its single worker and preserve the same bound.
StartFinal == /\ ~terminal /\ \A d \in Directions : call[d] = "idle"
              /\ native' = [d \in Directions |-> "blocked"]
              /\ call' = [d \in Directions |-> "waiting"]
              /\ UNCHANGED <<terminal, handlerReturned, attempts, orphaned, observed, terminalSuccess>>
Cancel == /\ ~terminal /\ \E d \in Directions : call[d] = "waiting"
          /\ terminal' = TRUE
          /\ UNCHANGED <<native, call, handlerReturned, attempts, orphaned, observed, terminalSuccess>>
ReturnCanceled(d) == /\ terminal /\ call[d] = "waiting"
                     /\ call' = [call EXCEPT ![d] = "returned"]
                     /\ observed' = TRUE
                     /\ UNCHANGED <<native, terminal, handlerReturned, attempts, orphaned, terminalSuccess>>
Retry == /\ terminal /\ attempts < 3
         /\ attempts' = attempts + 1
         /\ orphaned' = IF Mode = "unbounded" THEN orphaned + 1 ELSE orphaned
         /\ UNCHANGED <<native, call, terminal, handlerReturned, observed, terminalSuccess>>
ReturnHandler == /\ terminal /\ ~handlerReturned
                 /\ \A d \in Directions : call[d] # "waiting"
                 /\ handlerReturned' = TRUE
                 /\ terminalSuccess' = ~observed
                 /\ UNCHANGED <<native, call, terminal, attempts, orphaned, observed>>
NativeReturns(d) == /\ handlerReturned /\ native[d] = "blocked"
                    /\ native' = [native EXCEPT ![d] = "done"]
                    /\ UNCHANGED <<call, terminal, handlerReturned, attempts, orphaned, observed, terminalSuccess>>
Next == (\E d \in Directions : Start(d) \/ ReturnCanceled(d) \/ NativeReturns(d))
        \/ StartFinal \/ Cancel \/ Retry \/ ReturnHandler
Spec == Init /\ [][Next]_vars /\ WF_vars(Cancel) /\ WF_vars(ReturnHandler)
        /\ \A d \in Directions : WF_vars(ReturnCanceled(d)) /\ WF_vars(NativeReturns(d))
NoUnownedNativeWork == orphaned = 0
NoFalseSuccess == ~terminalSuccess
WorkersBounded == Cardinality({d \in Directions : native[d] = "blocked"}) + orphaned <= 2
CancellationReturns == terminal ~> (\A d \in Directions : call[d] # "waiting")
HandlerReclaimsWork == terminal ~> (\A d \in Directions : native[d] # "blocked")
=============================================================================
