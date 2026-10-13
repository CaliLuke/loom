-------------------------- MODULE CLIArguments --------------------------
EXTENDS Sequences, FiniteSets
CONSTANT Strategy
VARIABLES selected, multipart, interceptors, streaming
vars == <<selected, multipart, interceptors, streaming>>
Services == {1, 2}
Init == /\ selected \in {<<1>>, <<2>>, <<1, 2>>, <<2, 1>>}
        /\ multipart \in SUBSET Services
        /\ interceptors \in SUBSET Services
        /\ streaming \in SUBSET Services
Next == UNCHANGED vars
Entry(kind, service, members) ==
    IF service \in members THEN << <<kind, service>> >> ELSE <<>>
PerService(service) == Entry("multipart", service, multipart) \o
                       Entry("interceptors", service, interceptors)
Ordered(sel) == PerService(sel[1]) \o
               IF Len(sel) = 2 THEN PerService(sel[2]) ELSE <<>>
PerKind(kind, members, sel) == Entry(kind, sel[1], members) \o
                             IF Len(sel) = 2 THEN Entry(kind, sel[2], members) ELSE <<>>
Prefix(sel) == IF {sel[i] : i \in DOMAIN sel} \cap streaming = {}
              THEN <<>>
              ELSE << <<"dialer", 0>> >> \o PerKind("configurer", streaming, sel)
Declaration == Prefix(selected) \o Ordered(selected)
Call == CASE Strategy = "current" -> Prefix(selected) \o
              PerKind("multipart", multipart, selected) \o
              PerKind("interceptors", interceptors, selected)
        [] Strategy = "unfiltered" -> Prefix(<<1, 2>>) \o Ordered(<<1, 2>>)
        [] OTHER -> Prefix(selected) \o Ordered(selected)
MatchingIdentity == Declaration = Call
=============================================================================
