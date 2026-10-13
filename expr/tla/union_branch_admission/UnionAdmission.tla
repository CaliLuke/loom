------------------------ MODULE UnionAdmission ------------------------
EXTENDS Naturals, Sequences
CONSTANT Strategy
VARIABLES authored, phase, resolved, accepted, captured
vars == <<authored, phase, resolved, accepted, captured>>
Branch(owner, name, raw, tag) ==
  [owner |-> owner, name |-> name, raw |-> raw, tag |-> tag]
Designs == {
  <<Branch("u", "a", "a", "a"), Branch("u", "a", "a", "a")>>,
  <<Branch("u", "a:left", "a:left", "a"), Branch("u", "a:right", "a:right", "a")>>,
  <<Branch("u", "a", "left", "left"), Branch("u", "a", "right", "right")>>,
  <<Branch("u", "left", "left", "same"), Branch("u", "right", "right", "same")>>,
  <<Branch("u", "left", "left", "same"), Branch("u", "right", "right", "override")>>,
  <<Branch("u", "a", "a", "a"), Branch("v", "a", "a", "a")>>,
  <<Branch("u", "a", "a", "a"), Branch("u", "A", "A", "A")>>,
  <<Branch("u", "left", "left", "left"), Branch("u", "right", "right", "right")>>
}
Logical(n) == IF n \in {"a:left", "a:right"} THEN "a" ELSE n
NamesUnique(d) == \A i,j \in DOMAIN d : i # j /\ d[i].owner = d[j].owner =>
  Logical(d[i].name) # Logical(d[j].name)
TagsUnique(d) == \A i,j \in DOMAIN d : i # j /\ d[i].owner = d[j].owner => d[i].tag # d[j].tag
RawUnique(d) == \A i,j \in DOMAIN d : i # j /\ d[i].owner = d[j].owner => d[i].raw # d[j].raw
Valid(d) == NamesUnique(d) /\ TagsUnique(d)
GloballyUnique(d) == \A i,j \in DOMAIN d : i # j =>
  Logical(d[i].name) # Logical(d[j].name) /\ d[i].tag # d[j].tag
Admit(d) == CASE Strategy = "current" -> TRUE
             [] Strategy = "early" -> NamesUnique(d) /\ RawUnique(d)
             [] Strategy = "names" -> NamesUnique(d)
             [] Strategy = "tags" -> TagsUnique(d)
             [] Strategy = "global" -> GloballyUnique(d)
             [] Strategy = "owned" -> Valid(d)
Init == /\ authored \in Designs
        /\ phase = "authored"
        /\ resolved = <<>>
        /\ accepted = FALSE
        /\ captured = <<>>
Resolve == /\ phase = "authored"
           /\ resolved' = authored
           /\ phase' = "resolved"
           /\ UNCHANGED <<authored, accepted, captured>>
Validate == /\ phase = "resolved"
            /\ accepted' = Admit(resolved)
            /\ phase' = "validated"
            /\ UNCHANGED <<authored, resolved, captured>>
Capture == /\ phase = "validated"
           /\ captured' = IF accepted THEN resolved ELSE <<>>
           /\ phase' = "captured"
           /\ UNCHANGED <<authored, resolved, accepted>>
Next == Resolve \/ Validate \/ Capture
Spec == Init /\ [][Next]_vars
AdmissionSafe == phase \in {"validated", "captured"} /\ accepted => Valid(resolved)
ValidReuse == phase \in {"validated", "captured"} /\ Valid(resolved) => accepted
NoRewrite == phase = "captured" /\ accepted => captured = authored
=============================================================================
