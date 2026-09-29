------------------------ MODULE SchemaDeclaration ------------------------
EXTENDS Naturals
CONSTANTS Mode, SourceState
VARIABLES authored, sourceCursor, baseline, existingDeclaration,
          existingBaseline, phase, capturedTarget, liveTarget,
          selectedDeclaration, selectedContext, incomingRole, planned, cachedRole,
          byteVariant, variantContext, allocatedContext, existingContext, cachedContext,
          semanticDeclaration, capturedSemantic, acquisition
vars == <<authored, sourceCursor, baseline, existingDeclaration,
          existingBaseline, phase, capturedTarget, liveTarget,
          selectedDeclaration, selectedContext, incomingRole, planned, cachedRole,
          byteVariant, variantContext, allocatedContext, existingContext, cachedContext,
          semanticDeclaration, capturedSemantic, acquisition>>
Declarations == {"A", "B"}
PublicContext == "component/Public"
AllocatedContexts == {"component/Public_suffix1", "component/Public_suffix2"}
Contexts == AllocatedContexts \cup {PublicContext}

\* A noncanonical derived named component collides with an existing public
\* name. Source alias traversal can be at its original name, another authored
\* alias, or an unnamed structural node. Target ancestry is captured separately.
Init ==
    /\ acquisition \in {"fresh", "reuse"}
    /\ authored \in Declarations
    \* Explicit transport bodies can map to another semantic declaration while
    \* retaining their own authored definition and annotation authority.
    /\ semanticDeclaration \in Declarations
    /\ sourceCursor \in IF SourceState = "exhausted" THEN {""} ELSE Declarations \cup {""}
    /\ baseline \in {0, 1}
    /\ existingDeclaration \in Declarations \cup {""}
    /\ existingBaseline \in {0, 1}
    /\ incomingRole \in {"authored", "representation"}
    /\ planned \in BOOLEAN
    /\ incomingRole = "representation" => planned
    \* A shared child may first be reached through a different incoming edge.
    /\ cachedRole \in {"authored", "representation"}
    /\ allocatedContext \in AllocatedContexts
    \* This is the exact selected baseline binding, not a global name lookup.
    /\ existingContext \in Contexts
    \* A prior projected memo entry may have equal shape/declaration but a
    \* different authored annotation binding.
    /\ cachedContext \in Contexts
    /\ byteVariant \in BOOLEAN
    /\ variantContext = ""
    /\ phase = "copy"
    /\ capturedTarget = ""
    /\ capturedSemantic = ""
    /\ liveTarget = authored
    /\ selectedDeclaration = ""
    /\ selectedContext = ""

Capture ==
    /\ phase = "copy"
    /\ capturedTarget' = IF Mode = "mapping-confused" THEN semanticDeclaration ELSE authored
    /\ capturedSemantic' = IF Mode = "mapping-lost" THEN authored ELSE semanticDeclaration
    /\ phase' = "rename"
    /\ UNCHANGED <<acquisition, authored, sourceCursor, baseline, existingDeclaration,
                    existingBaseline, liveTarget, selectedDeclaration, selectedContext,
                    incomingRole, planned, cachedRole, byteVariant, variantContext,
                    allocatedContext, existingContext, cachedContext, semanticDeclaration>>
Rename ==
    /\ phase = "rename"
    /\ liveTarget' = "TransportName"
    /\ phase' = "query"
    /\ UNCHANGED <<acquisition, authored, sourceCursor, baseline, existingDeclaration,
                    existingBaseline, capturedTarget, selectedDeclaration, selectedContext,
                    incomingRole, planned, cachedRole, byteVariant, variantContext,
                    allocatedContext, existingContext, cachedContext, semanticDeclaration,
                    capturedSemantic>>

QueryDeclaration ==
    IF Mode = "legacy"
    THEN IF sourceCursor = "" THEN liveTarget ELSE sourceCursor
    ELSE IF Mode = "source-fallback"
         THEN IF sourceCursor = "" THEN capturedTarget ELSE sourceCursor
         ELSE capturedTarget
ReuseExistingContext ==
    IF Mode \in {"legacy", "target-only"} THEN FALSE
    ELSE IF Mode = "unchecked-sharing"
         THEN existingDeclaration # "" /\ existingBaseline = baseline
         ELSE /\ existingDeclaration = QueryDeclaration
              /\ existingDeclaration # ""
              /\ existingBaseline = baseline
              /\ CASE Mode = "plan-only" -> planned
                   [] Mode = "child-cached" -> cachedRole = "representation"
                   [] Mode \in {"edge-checked", "byte-canonical", "fingerprint-cached"}
                        -> incomingRole = "representation"
                   [] OTHER -> TRUE
BoundContext ==
    IF acquisition = "reuse" THEN existingContext
    ELSE IF Mode \in {"allocation-owned", "target-only", "byte-canonical",
                      "fingerprint-cached", "mapping-confused", "mapping-lost"}
         THEN allocatedContext
    ELSE IF ~ReuseExistingContext THEN allocatedContext
    ELSE IF Mode \in {"edge-checked", "byte-canonical", "fingerprint-cached"}
         THEN existingContext ELSE PublicContext
Query ==
    /\ phase = "query"
    /\ selectedDeclaration' = QueryDeclaration
    /\ selectedContext' = BoundContext
    /\ phase' = "project"
    /\ UNCHANGED <<acquisition, authored, sourceCursor, baseline, existingDeclaration,
                    existingBaseline, capturedTarget, liveTarget,
                    incomingRole, planned, cachedRole, byteVariant, variantContext,
                    allocatedContext, existingContext, cachedContext, semanticDeclaration,
                    capturedSemantic>>
Project ==
    /\ phase = "project"
    /\ variantContext' =
        IF byteVariant /\ Mode = "fingerprint-cached"
           /\ existingDeclaration = selectedDeclaration /\ existingBaseline = baseline
        THEN cachedContext
        ELSE IF byteVariant /\ Mode = "byte-canonical"
             THEN PublicContext ELSE selectedContext
    /\ phase' = "done"
    /\ UNCHANGED <<acquisition, authored, sourceCursor, baseline, existingDeclaration,
                    existingBaseline, capturedTarget, liveTarget, selectedDeclaration,
                    selectedContext, incomingRole, planned, cachedRole, byteVariant,
                    allocatedContext, existingContext, cachedContext, semanticDeclaration,
                    capturedSemantic>>
Next == Capture \/ Rename \/ Query \/ Project
Spec == Init /\ [][Next]_vars

\* These obligations refer to the authored target before any rename, not the
\* query result or the source cursor. Shapes are exact opaque baseline labels.
DeclarationAuthority == phase = "done" => selectedDeclaration = authored
CapturedIdentityStable == phase # "copy" => capturedTarget = authored
SemanticBindingStable == phase # "copy" => capturedSemantic = semanticDeclaration
VariantOwnerPreserved == phase = "done" => variantContext = selectedContext
\* Independent allocator outcome: a fresh allocation owns its new context;
\* actual reuse retains the selected existing binding. Pairing roles do not
\* choose this outcome. The Go naming function runs only on the fresh branch.
AcquiredOwnerPreserved == phase = "done" =>
    selectedContext = IF acquisition = "reuse" THEN existingContext ELSE allocatedContext
=============================================================================
