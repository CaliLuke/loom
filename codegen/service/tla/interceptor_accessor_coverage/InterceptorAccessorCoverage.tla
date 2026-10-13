---------------- MODULE InterceptorAccessorCoverage ----------------
EXTENDS Naturals, Sequences, FiniteSets
CONSTANT Policy
VARIABLES server, client, declarations, serverCalls, clientCalls, built
vars == <<server, client, declarations, serverCalls, clientCalls, built>>
Methods == {1, 2, 3}
Ordered(s) == SelectSeq(<<1, 2, 3>>, LAMBDA m: m \in s)
Members(seq) == {seq[i]: i \in DOMAIN seq}

Init == /\ server \in SUBSET Methods
        /\ client \in SUBSET Methods
        /\ server \cup client # {}
        /\ declarations = <<>>
        /\ serverCalls = {}
        /\ clientCalls = {}
        /\ built = FALSE

Build == /\ ~built
         /\ declarations' = CASE Policy = "current" ->
                                      Ordered(IF server # {} THEN server ELSE client)
                                  [] Policy = "append" -> Ordered(server) \o Ordered(client)
                                  [] OTHER -> Ordered(server \cup client)
         /\ serverCalls' = IF Policy = "broaden" THEN server \cup client ELSE server
         /\ clientCalls' = IF Policy = "broaden" THEN server \cup client ELSE client
         /\ built' = TRUE
         /\ UNCHANGED <<server, client>>
Next == Build \/ (built /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
Coverage == built => Members(declarations) = server \cup client
UniqueDeclarations == built => Len(declarations) = Cardinality(Members(declarations))
SideLocalCalls == built => serverCalls = server /\ clientCalls = client
StableOrder == built => declarations = Ordered(Members(declarations))
=============================================================================
