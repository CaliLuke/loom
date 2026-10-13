-------------------------- MODULE Admission --------------------------
EXTENDS Naturals
CONSTANT Mode
VARIABLES phase, sourceNullable, anyMessage, localNullable, implicit,
          inspected, admitted
vars == <<phase, sourceNullable, anyMessage, localNullable, implicit,
          inspected, admitted>>
Init == /\ phase = "prepared"
        /\ sourceNullable \in BOOLEAN
        /\ anyMessage \in BOOLEAN
        /\ localNullable \in BOOLEAN
        /\ implicit \in BOOLEAN
        /\ (implicit => ~localNullable)
        /\ inspected = FALSE
        /\ admitted = FALSE
\* Prepared transport placeholders lack completed service projections.
Validate == /\ phase = "prepared"
            /\ inspected' = IF Mode = "legacy"
                              THEN ~implicit /\ localNullable
                              ELSE sourceNullable \/ localNullable
            /\ admitted' = (anyMessage \/ ~inspected')
            /\ phase' = "validated"
            /\ UNCHANGED <<sourceNullable, anyMessage, localNullable, implicit>>
Finalize == /\ phase = "validated"
            /\ phase' = "finalized"
            /\ UNCHANGED <<sourceNullable, anyMessage, localNullable, implicit,
                           inspected, admitted>>
Next == Validate \/ Finalize
Spec == Init /\ [][Next]_vars
NoLostPresence == phase = "finalized" /\ admitted
                  => anyMessage \/ ~(sourceNullable \/ localNullable)
NativeAdmitted == phase # "prepared" /\ ~sourceNullable /\ ~localNullable
                  => admitted
AnyAdmitted == phase # "prepared" /\ anyMessage => admitted
=====================================================================
