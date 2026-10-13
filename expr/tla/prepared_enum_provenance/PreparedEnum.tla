---------------------------- MODULE PreparedEnum ----------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
VARIABLES phase, declaration, members, sourceMembers, defaultValue, minimum, wireRenamed
vars == <<phase, declaration, members, sourceMembers, defaultValue, minimum, wireRenamed>>
Consumers == {1, 2}
OriginalMembers == {1, 2}
Admitted(values, bound) == {n \in values : n >= bound}
Init == /\ phase = [c \in Consumers |-> "source"]
        /\ declaration = [c \in Consumers |-> "base"]
        /\ members = [c \in Consumers |-> OriginalMembers]
        /\ sourceMembers = OriginalMembers
        /\ defaultValue \in {1, 2, 3}
        /\ minimum \in {1, 2}
        /\ wireRenamed \in BOOLEAN
Map(c) == /\ phase[c] = "source"
          /\ phase' = [phase EXCEPT ![c] = "mapped"]
          /\ declaration' = [declaration EXCEPT ![c] =
               IF Mode \in {"copy", "lower"} THEN "physical" ELSE "base"]
          /\ members' = [members EXCEPT ![c] = IF Mode = "first-local" THEN {1, 2, 3} ELSE sourceMembers]
          /\ UNCHANGED <<sourceMembers, defaultValue, minimum, wireRenamed>>
Wrap(c) == /\ phase[c] = "mapped"
           /\ phase' = [phase EXCEPT ![c] = "wrapped"]
           /\ UNCHANGED <<declaration, members, sourceMembers, defaultValue, minimum, wireRenamed>>
Mutate(c) == /\ phase[c] = "wrapped"
             /\ phase' = [phase EXCEPT ![c] = "mutated"]
             /\ members' = [members EXCEPT ![c] = {2}]
             /\ sourceMembers' = IF Mode = "borrow" THEN {2} ELSE sourceMembers
             /\ UNCHANGED <<declaration, defaultValue, minimum, wireRenamed>>
Next == (\E c \in Consumers : Map(c) \/ Wrap(c) \/ Mutate(c)) \/ UNCHANGED vars
DeclarationCheck(c) == members[c] \subseteq
  (IF declaration[c] = "base" THEN {1, 2, 3} ELSE {n \in {1, 2, 3} : n >= minimum})
EnumIdentity == \A c \in Consumers : phase[c] \in {"mapped", "wrapped"} => declaration[c] = "base"
EnumConstantsRemainValid == \A c \in Consumers : phase[c] \in {"mapped", "wrapped"} => DeclarationCheck(c)
EffectiveMembership == \A c \in Consumers : phase[c] \in {"mapped", "wrapped"} =>
  Admitted(members[c], minimum) = Admitted(OriginalMembers, minimum)
TargetName == IF wireRenamed THEN "n:wire_n" ELSE "n"
ProjectedFields(c) == IF Mode = "full-name" /\ TargetName # "n" THEN {} ELSE {"n"}
ProjectionRetainsLogicalMember == \A c \in Consumers : phase[c] \in {"mapped", "wrapped"} => ProjectedFields(c) = {"n"}
SourceUnchanged == sourceMembers = OriginalMembers
DefaultAccepted(c) == IF Mode = "default-at-declaration"
                     THEN defaultValue \in members[c]
                     ELSE defaultValue \in Admitted(members[c], minimum)
DefaultsUseFinalContract == \A c \in Consumers : phase[c] = "wrapped" =>
  DefaultAccepted(c) = (defaultValue \in Admitted(members[c], minimum))
Spec == Init /\ [][Next]_vars
=============================================================================
