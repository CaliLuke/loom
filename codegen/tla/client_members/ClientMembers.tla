------------------------ MODULE ClientMembers ------------------------
EXTENDS Naturals, Sequences, FiniteSets
CONSTANT Strategy
VARIABLES specimen, index, names
vars == <<specimen, index, names>>
Specimens == {"fixed", "websocket", "doers", "builders", "streams", "ordinary"}
Preferred(s) == CASE s = "fixed" -> <<"RestoreResponseBody", "RestoreResponseBodyEndpoint">>
                [] s = "websocket" -> <<"Close", "IsClosed", "CloseEndpoint">>
                [] s = "doers" -> <<"Read", "ReadDoer", "ReadDoer">>
                [] s = "builders" -> <<"Read", "BuildReadRequest", "BuildReadRequest">>
                [] s = "streams" -> <<"Watch", "WatchStream", "WatchStream">>
                [] OTHER -> <<"Close", "IsClosed", "ReadDoer", "BuildReadRequest">>
Fixed(s) == IF s = "websocket" THEN {"Doer", "RestoreResponseBody", "Close", "IsClosed"}
            ELSE {"RestoreResponseBody"}
Used == {names[i] : i \in DOMAIN names}
Candidate(p, n) == CASE n = 0 -> p [] n = 1 -> p \o "Endpoint"
                  [] n = 2 -> p \o "Endpoint2" [] OTHER -> p \o "Endpoint3"
ChooseName(p) == LET available == {n \in 0..3 : Candidate(p,n) \notin (Fixed(specimen) \cup Used)}
                    first == CHOOSE n \in available : \A m \in available : n <= m
                IN Candidate(p, first)
Init == /\ specimen \in Specimens /\ index = 1 /\ names = <<>>
Allocate == /\ index <= Len(Preferred(specimen))
            /\ names' = Append(names, IF Strategy = "current" \/
                 (Strategy = "fixed_only" /\ Preferred(specimen)[index] \notin Fixed(specimen))
                 THEN Preferred(specimen)[index] ELSE ChooseName(Preferred(specimen)[index]))
            /\ index' = index + 1 /\ UNCHANGED specimen
Next == Allocate \/ (index > Len(Preferred(specimen)) /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
UniqueMembers == /\ \A i,j \in DOMAIN names : i # j => names[i] # names[j]
                 /\ Used \cap Fixed(specimen) = {}
StableOrdinary == specimen = "ordinary" => names = SubSeq(Preferred(specimen),1,index-1)
=============================================================================
