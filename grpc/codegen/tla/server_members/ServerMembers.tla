------------------------- MODULE ServerMembers -------------------------
EXTENDS Naturals, Sequences, FiniteSets
CONSTANT Strategy
VARIABLES specimen, phase, names, references
vars == <<specimen, phase, names, references>>
Specimens == {"ordinary", "handler", "fallback", "embedded", "stream"}
RPCs(s) == CASE s = "handler" -> {"Mount", "MountH"}
           [] s = "fallback" -> {"Mount", "MountH", "MountHHandler", "MountHHandler2"}
           [] s = "embedded" -> {"UnimplementedMembersServer"}
           [] s = "stream" -> {"Mount", "MountH"}
           [] OTHER -> {"Read", "Write"}
Preferred(s) == CASE s = "embedded" -> <<"UnimplementedMembersServer", "UnimplementedMembersServerH">>
                [] s = "ordinary" -> <<"UnimplementedMembersServer", "ReadH", "WriteH">>
                [] s = "fallback" -> <<"UnimplementedMembersServer", "MountH", "MountHH", "MountHHandlerH", "MountHHandler2H">>
                [] OTHER -> <<"UnimplementedMembersServer", "MountH", "MountHH">>
Used == {names[i] : i \in DOMAIN names}
Candidate(p, n) == CASE n = 0 -> p [] n = 1 -> p \o "Handler"
                  [] n = 2 -> p \o "Handler2" [] OTHER -> p \o "Handler3"
Choose(p) == LET available == {n \in 0..3 : Candidate(p,n) \notin (Used \cup RPCs(specimen))}
                first == CHOOSE n \in available : \A m \in available : n <= m
            IN Candidate(p, first)
Init == /\ specimen \in Specimens /\ phase = 1 /\ names = <<>> /\ references = <<>>
Allocate == /\ phase <= Len(Preferred(specimen))
            /\ LET p == Preferred(specimen)[phase]
                   chosen == IF Strategy = "current" \/ (Strategy = "handlers-only" /\ phase = 1)
                             THEN p ELSE Choose(p)
               IN /\ names' = Append(names, chosen)
                  /\ references' = Append(references, IF Strategy = "declarations-only" THEN p ELSE chosen)
            /\ phase' = phase + 1 /\ UNCHANGED specimen
Next == Allocate \/ (phase > Len(Preferred(specimen)) /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
UniqueMembers == /\ Used \cap RPCs(specimen) = {}
                 /\ Cardinality(Used) = Len(names)
BindingConsistency == names = references
StableOrdinary == specimen = "ordinary" => names = SubSeq(Preferred(specimen),1,phase-1)
=============================================================================
