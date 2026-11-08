---------------------- MODULE ParentHintProjection ----------------------
EXTENDS Naturals, Sequences, FiniteSets
CONSTANT Mode
VARIABLES phase, supported, strong, partial, retained, emitted, hint,
          originalHint, otherOperationHint, omitted, blocked, sourceAdmitted
vars == <<phase, supported, strong, partial, retained, emitted, hint,
          originalHint, otherOperationHint, omitted, blocked, sourceAdmitted>>
Children == {1, 2}
Targets == {0, 1, 2} \* 0 stands for a non-child default target.
NamedChildren == {1} \* Child 2 is an inline occurrence, retained only by a selected use.
SupportedChildren == supported \cap NamedChildren
ChildEvidence == (SupportedChildren \cup strong) \cap Children
StrongFailure == (strong \ supported) # {}
NeedsOmission == partial /\ ChildEvidence = {}
Init == /\ phase = "project"
        /\ supported \in SUBSET Targets
        /\ strong \in SUBSET Targets
        /\ partial \in BOOLEAN
        /\ retained = {}
        /\ emitted = {}
        /\ hint = TRUE
        /\ originalHint = TRUE
        /\ otherOperationHint = TRUE
        /\ omitted = FALSE
        /\ blocked = FALSE
        /\ sourceAdmitted = (Mode # "source-selected" \/ ChildEvidence # {})

Project == /\ phase = "project"
           /\ retained' = CASE Mode = "pruned" -> strong
                              [] Mode = "strong-all" -> Children \cup strong
                              [] OTHER -> SupportedChildren \cup strong
           /\ blocked' = CASE Mode = "strong-all" -> ((Children \ supported) # {} \/ StrongFailure)
                             [] Mode = "pruned" -> StrongFailure
                             [] Mode = "named-only" -> (StrongFailure \/ (~partial /\ SupportedChildren = {}))
                             [] OTHER -> (StrongFailure \/ (~partial /\ ChildEvidence = {}))
           /\ hint' = CASE Mode = "pruned" -> TRUE
                           [] Mode = "named-only" -> ~(partial /\ SupportedChildren = {})
                           [] OTHER -> ~NeedsOmission
           /\ omitted' = (Mode # "unreported" /\ ~hint')
           /\ originalHint' = IF Mode = "shared" THEN hint' ELSE originalHint
           /\ otherOperationHint' = IF Mode = "shared" THEN hint' ELSE otherOperationHint
           /\ phase' = "export"
           /\ UNCHANGED <<supported, strong, partial, emitted, sourceAdmitted>>

Export == /\ phase = "export"
          /\ emitted' = IF Mode = "drop-unused" THEN strong ELSE retained
          /\ phase' = "done"
          /\ UNCHANGED <<supported, strong, partial, retained, hint,
                          originalHint, otherOperationHint, omitted, blocked, sourceAdmitted>>
Next == Project \/ Export
Spec == Init /\ [][Next]_vars
ProjectedHintDefined == phase = "done" /\ ~blocked /\ hint => (emitted \cap Children) # {}
SupportedChildrenRetained == phase = "done" /\ ~blocked => SupportedChildren \subseteq retained
ValidationIndependent == phase = "done" /\ ~StrongFailure /\ ChildEvidence # {} => ~blocked
StrongDependencies == phase = "done" /\ StrongFailure => blocked
RetainedEvidenceKeepsHint == phase = "done" /\ ~blocked /\ ChildEvidence # {} => hint
OmissionReported == phase = "done" => (omitted = ~hint)
SourceAdmissionIndependent == sourceAdmitted
SourceIsolated == originalHint /\ otherOperationHint
PublicationComplete == phase = "done" /\ ~blocked => emitted = retained
\* A stable order is derived from fixed source identities, never traversal order.
Ordered == (IF 0 \in retained THEN <<0>> ELSE <<>>) \o
           (IF 1 \in retained THEN <<1>> ELSE <<>>) \o
           (IF 2 \in retained THEN <<2>> ELSE <<>>)
OrderStable == Len(Ordered) = Cardinality(retained) /\
               (\A i, j \in 1..Len(Ordered): i < j => Ordered[i] < Ordered[j])
=============================================================================
