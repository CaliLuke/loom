------------------------ MODULE DeclarationMetadata ------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Policy
VARIABLES uses, invalidDeclaration, invalidLocal, declarationErrors, localErrors, done
vars == <<uses, invalidDeclaration, invalidLocal, declarationErrors, localErrors, done>>
Init == /\ uses \in 0..2
        /\ invalidDeclaration \in BOOLEAN
        /\ invalidLocal \in SUBSET (1..uses)
        /\ declarationErrors = 0
        /\ localErrors = {}
        /\ done = FALSE
Validate == /\ ~done
            /\ declarationErrors' = IF invalidDeclaration
                  THEN IF Policy = "occurrence" THEN uses ELSE 1
                  ELSE 0
            /\ localErrors' = IF Policy = "global" /\ invalidDeclaration
                              THEN {} ELSE invalidLocal
            /\ done' = TRUE
            /\ UNCHANGED <<uses, invalidDeclaration, invalidLocal>>
Spec == Init /\ [][Validate]_vars
DeclarationOnce == done => declarationErrors = IF invalidDeclaration THEN 1 ELSE 0
LocalPreserved == done => localErrors = invalidLocal
=============================================================================
