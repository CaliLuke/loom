---------------------- MODULE ErrorDeclarations ----------------------
EXTENDS Naturals, Sequences, FiniteSets
CONSTANT Strategy
VARIABLES authored, phase, accepted, emitted
vars == <<authored, phase, accepted, emitted>>
Entry(i, s, n) == [id |-> i, scope |-> s, name |-> n]
Designs == {
  <<Entry(1, "api", "boom"), Entry(2, "api", "boom")>>,
  <<Entry(1, "service", "boom"), Entry(2, "service", "boom")>>,
  <<Entry(1, "method", "boom"), Entry(2, "method", "boom")>>,
  <<Entry(1, "service", "boom"), Entry(2, "service", "boom"), Entry(3, "method", "boom")>>,
  <<Entry(1, "api", "boom"), Entry(2, "service", "boom"), Entry(3, "method", "boom")>>,
  <<Entry(1, "service", "foo-bar"), Entry(2, "service", "foo_bar")>>,
  <<Entry(1, "service", "boom"), Entry(2, "service", "BOOM")>>,
  <<Entry(1, "method", "only")>>,
  <<Entry(1, "service", "other"), Entry(2, "method", "local")>>,
  <<>>
}
LocalUnique(d) == \A i, j \in DOMAIN d :
  i # j => (d[i].scope # d[j].scope \/ d[i].name # d[j].name)
NamesUnique(d) == \A i, j \in DOMAIN d : i # j => d[i].name # d[j].name
HasMethod(d, n) == \E i \in DOMAIN d : d[i].scope = "method" /\ d[i].name = n
FirstService(d, e) == ~\E i \in DOMAIN d :
  d[i].scope = "service" /\ d[i].name = e.name /\ d[i].id < e.id
Effective(d) == SelectSeq(d, LAMBDA e: e.scope = "method") \o
  SelectSeq(d, LAMBDA e:
    e.scope = "service" /\ ~HasMethod(d, e.name) /\ FirstService(d, e))
Admit(d) == CASE Strategy = "current" -> TRUE
             [] Strategy = "effective" -> NamesUnique(Effective(d))
             [] Strategy = "global" -> NamesUnique(d)
             [] Strategy = "local" -> LocalUnique(d)
Init == /\ authored \in Designs
        /\ phase = "authored"
        /\ accepted = FALSE
        /\ emitted = <<>>
Validate == /\ phase = "authored"
            /\ accepted' = Admit(authored)
            /\ phase' = "validated"
            /\ UNCHANGED <<authored, emitted>>
Finalize == /\ phase = "validated"
            /\ phase' = "finished"
            /\ emitted' = IF accepted THEN Effective(authored) ELSE <<>>
            /\ UNCHANGED <<authored, accepted>>
Next == Validate \/ Finalize
Spec == Init /\ [][Next]_vars
RejectLocalDuplicates == phase # "authored" /\ accepted => LocalUnique(authored)
AcceptValidReuse == phase # "authored" /\ LocalUnique(authored) => accepted
PreserveSelection == phase = "finished" /\ accepted => emitted = Effective(authored)
=============================================================================
