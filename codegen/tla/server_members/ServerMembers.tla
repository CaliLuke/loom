------------------------ MODULE ServerMembers ------------------------
EXTENDS Naturals, Sequences, FiniteSets
CONSTANT Strategy
VARIABLES specimen, index, names
vars == <<specimen, index, names>>
Specimens == {"http", "rpc", "websocket", "roles", "keywords", "ordinary"}
Preferred(s) == CASE s = "http" -> <<"Mount", "MountHandler", "Mounts", "Read", "Read">>
                [] s = "rpc" -> <<"Methods", "Handler", "MethodNames">>
                [] s = "websocket" -> <<"decoder", "conn", "decoderHandler">>
                [] s = "roles" -> <<"read", "readEndpoint", "readEndpoint">>
                [] s = "keywords" -> <<"type", "func">>
                [] OTHER -> <<"Read", "Write">>
ServerFixed(s) == CASE s = "http" -> {"Mount", "Service", "Use", "Mounts", "MethodNames"}
                  [] s = "rpc" -> {"Handler", "Methods", "MethodNames", "Mount", "Service"}
                  [] s = "websocket" -> {"decoder", "encoder", "errhandler"}
                  [] s = "roles" -> {"decoder", "encoder", "errhandler"}
                  [] s = "keywords" -> {}
                  [] OTHER -> {"Mount", "Service", "Use"}
CopiedReceiverFixed(s) == IF s = "websocket" THEN {"conn", "cancel", "r", "w"} ELSE {}
Used == {names[i] : i \in DOMAIN names}
Candidate(p, n) == CASE n = 0 -> p [] n = 1 -> p \o "Handler"
                  [] n = 2 -> p \o "Handler2" [] OTHER -> p \o "Handler3"
Keywords == {"type", "func"}
Reserved == Keywords \cup ServerFixed(specimen) \cup
            IF Strategy = "owned" THEN CopiedReceiverFixed(specimen) ELSE {}
ChooseName(p) == LET available == {n \in 0..3 : Candidate(p,n) \notin (Reserved \cup Used)}
                    first == CHOOSE n \in available : \A m \in available : n <= m
                IN Candidate(p, first)
Init == /\ specimen \in Specimens /\ index = 1 /\ names = <<>>
Allocate == /\ index <= Len(Preferred(specimen))
            /\ names' = Append(names, IF Strategy = "current" THEN Preferred(specimen)[index]
                                     ELSE ChooseName(Preferred(specimen)[index]))
            /\ index' = index + 1 /\ UNCHANGED specimen
Next == Allocate \/ (index > Len(Preferred(specimen)) /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
UniqueMembers == /\ \A i,j \in DOMAIN names : i # j => names[i] # names[j]
                 /\ Used \cap (Keywords \cup ServerFixed(specimen) \cup CopiedReceiverFixed(specimen)) = {}
StableOrdinary == specimen = "ordinary" => names = SubSeq(Preferred(specimen),1,index-1)
=============================================================================
