----------------------- MODULE HandlerDispatch -----------------------
EXTENDS Naturals, Sequences
CONSTANT Mode
VARIABLES chain, mounted, captured, served, selected, endpoint, reject
vars == <<chain, mounted, captured, served, selected, endpoint, reject>>
Init == /\ chain = <<>> /\ mounted = FALSE /\ captured = <<>>
        /\ served = FALSE /\ selected = <<>> /\ endpoint = FALSE
        /\ reject \in BOOLEAN
Use == /\ ~served /\ Len(chain) < 2
       /\ chain' = <<Len(chain) + 1>> \o chain
       /\ UNCHANGED <<mounted, captured, served, selected, endpoint, reject>>
Mount == /\ ~mounted /\ ~served
         /\ mounted' = TRUE /\ captured' = chain
         /\ UNCHANGED <<chain, served, selected, endpoint, reject>>
Serve == /\ ~served /\ mounted /\ Len(chain) = 2
         /\ selected' = IF Mode = "legacy" THEN captured ELSE chain
         /\ endpoint' = (~reject \/ Len(IF Mode = "legacy" THEN captured ELSE chain) = 0)
         /\ served' = TRUE
         /\ UNCHANGED <<chain, mounted, captured, reject>>
Next == Use \/ Mount \/ Serve \/ (served /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
CurrentChain == served => selected = chain
MiddlewareOrder == served => selected = <<2,1>>
RejectBeforeEndpoint == served => endpoint = ~reject
======================================================================
