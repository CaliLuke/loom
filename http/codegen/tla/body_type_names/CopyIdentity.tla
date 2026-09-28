------------------------ MODULE CopyIdentity ------------------------
EXTENDS Naturals
CONSTANT Mode
VARIABLES copied, branchShape, branchName
vars == <<copied, branchShape, branchName>>
RootName == "OtherRequestBody"
RootID == IF Mode = "legacy" THEN RootName ELSE "svc#OtherRequestBody"
RootKey == <<Mode = "checked", RootID>>
BranchKey == <<FALSE, branchName>>
Init == /\ copied = FALSE /\ branchShape = "object"
        /\ branchName \in {RootName, "svc#OtherRequestBody"}
Copy == /\ ~copied
        /\ copied' = TRUE
        /\ branchShape' = IF RootKey = BranchKey THEN "union" ELSE "object"
        /\ UNCHANGED branchName
Next == Copy \/ (copied /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
BranchPreserved == branchShape = "object"
Terminates == <>copied
=====================================================================
