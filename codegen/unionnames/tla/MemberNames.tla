-------------------------- MODULE MemberNames --------------------------
EXTENDS Naturals, Sequences, FiniteSets, TLC
CONSTANT Policy
VARIABLE specimen
Fixed == {"Kind", "Validate", "MarshalJSON", "UnmarshalJSON", "MarshalFormValues", "UnmarshalFormValues"}
Cases == {<<"Kind", "Kind2", "Text", "AsText", "SetText">>,
          <<"Text", "Text", "Text2">>, <<"Validate", "MarshalJSON">>,
          <<"Alpha", "Beta">>}
Values(s) == {s[i] : i \in 1..Len(s)}
Fresh(base, used) == LET n == CHOOSE i \in 2..20 :
                               base \o ToString(i) \notin used /\
                               \A j \in 2..(i-1) : base \o ToString(j) \in used
                    IN base \o ToString(n)
RECURSIVE Methods(_, _)
Methods(s, i) == IF i = 0 THEN <<>> ELSE
  LET previous == Methods(s, i-1)
      name == IF s[i] \notin Values(previous) THEN s[i]
              ELSE Fresh(s[i], Values(s) \cup Values(previous))
  IN Append(previous, name)
Reserved(m) == Fixed \cup {"As" \o n : n \in Values(m)} \cup {"Set" \o n : n \in Values(m)}
RECURSIVE Fields(_, _)
Fields(m, i) == IF i = 0 THEN <<>> ELSE
  LET previous == Fields(m, i-1)
      name == IF m[i] \notin Reserved(m) THEN m[i]
              ELSE Fresh(m[i], Reserved(m) \cup Values(m) \cup Values(previous))
  IN Append(previous, name)
M == IF Policy = "old" THEN specimen ELSE Methods(specimen, Len(specimen))
F == IF Policy = "old" THEN specimen ELSE Fields(M, Len(M))
Init == specimen \in Cases
Next == UNCHANGED specimen
UniqueMethods == Cardinality(Values(M)) = Len(M)
UniqueFields == Cardinality(Values(F)) = Len(F)
Disjoint == Values(F) \cap Reserved(M) = {}
OrdinaryStable == specimen = <<"Alpha", "Beta">> => F = specimen /\ M = specimen
Spec == Init /\ [][Next]_specimen
=============================================================================
