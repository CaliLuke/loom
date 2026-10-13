------------------------ MODULE WebSocketAdmission ------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Policy
VARIABLES input, phase, admitted
vars == <<input, phase, admitted>>
Kinds == {"none", "ordinary-header", "explicit-header", "implicit-bearer",
          "implicit-basic", "session-cookie", "message-token"}
Credential(k) == k \notin {"none", "ordinary-header"}
Forbidden(k) == k \notin {"none", "message-token"}
RawMapped(k) == k \in {"ordinary-header", "explicit-header"}
ResolvedMapped(k) == Forbidden(k) /\ k # "implicit-basic"
Safe(i) == ~i.ws \/ (~Forbidden(i.kind) /\
           (~Credential(i.kind) \/ i.source = "same"))
Inputs == [kind : Kinds, ws : BOOLEAN, source : {"same", "lost", "retagged"}]
Init == /\ input \in Inputs
        /\ phase = 0
        /\ admitted = FALSE
Resolve == /\ phase = 0
           /\ phase' = 1
           /\ UNCHANGED <<input, admitted>>
Validate == /\ phase = 1
            /\ admitted' = CASE ~input.ws -> TRUE
                  [] Policy = "legacy" -> ~RawMapped(input.kind)
                  [] Policy = "resolved-mappings" -> ~ResolvedMapped(input.kind)
                  [] Policy = "location-only" -> ~Forbidden(input.kind)
                  [] Policy = "blanket-security" -> ~Forbidden(input.kind) /\ ~Credential(input.kind)
                  [] OTHER -> Safe(input)
            /\ phase' = 2
            /\ UNCHANGED input
Finalize == /\ phase = 2
            /\ phase' = 3
            /\ UNCHANGED <<input, admitted>>
Next == Resolve \/ Validate \/ Finalize
NoUnsupportedWireBinding == phase < 2 \/ ~admitted \/ ~input.ws \/ ~Forbidden(input.kind)
NoLostCredential == phase < 3 \/ ~admitted \/ ~input.ws \/
                    ~Credential(input.kind) \/ input.source = "same"
PreserveSupported == phase < 2 \/ ~Safe(input) \/ admitted
Spec == Init /\ [][Next]_vars
=============================================================================
