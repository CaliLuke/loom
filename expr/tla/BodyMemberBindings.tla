-------------------------- MODULE BodyMemberBindings --------------------------
EXTENDS Naturals, FiniteSets

CONSTANT Mode
VARIABLES ready, binding, declarationBinding
vars == <<ready, binding, declarationBinding>>

\* Two endpoint occurrences reuse one authored body declaration. HTTP's
\* finalized mapping already identifies each selected service member.
Endpoints == {1, 2}
Members == {1, 2}
Sources == Endpoints \X Members
NoSource == <<0, 0>>
Authored == [m \in Members |-> NoSource]
Expected(e) == [m \in Members |-> <<e, m>>]

Init ==
    /\ ready = {}
    /\ binding = [e \in Endpoints |-> Authored]
    /\ declarationBinding = Authored

Finalize(e) ==
    /\ e \notin ready
    /\ ready' = ready \cup {e}
    /\ IF Mode = "shared-declaration"
          THEN /\ declarationBinding' = Expected(e)
               /\ UNCHANGED binding
          ELSE /\ binding' = [binding EXCEPT ![e] =
                      IF Mode = "root-only" THEN Authored ELSE Expected(e)]
               /\ UNCHANGED declarationBinding

Next == \E e \in Endpoints : Finalize(e)
Spec == Init /\ [][Next]_vars

Observed(e) == IF Mode = "shared-declaration"
              THEN declarationBinding ELSE binding[e]

\* Expected is input correspondence, independent of the stored binding.
MemberCorrespondence == \A e \in ready : Observed(e) = Expected(e)
DeclarationUnchanged == declarationBinding = Authored
TypeOK ==
    /\ ready \subseteq Endpoints
    /\ binding \in [Endpoints -> [Members -> Sources \cup {NoSource}]]
    /\ declarationBinding \in [Members -> Sources \cup {NoSource}]
=============================================================================
