-------------------------- MODULE ByteProjection --------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
VARIABLES bytesEnum, textEnum, matches, output, phase
vars == <<bytesEnum,textEnum,matches,output,phase>>
Allowed == {{"raw"}, {"wire"}, {"raw","wire"}}
Init == /\ bytesEnum \in Allowed /\ textEnum \in Allowed
        /\ matches = {} /\ output = "none" /\ phase = "select"
WireMatches(value) == {branch \in {"bytes","text"} :
                       IF branch = "bytes" THEN value \in bytesEnum ELSE value \in textEnum}
Select == /\ phase = "select"
          /\ matches' = IF Mode = "legacy" THEN {}
                        ELSE IF Mode = "per-branch"
                        THEN {branch \in {"bytes","text"} :
                              IF branch = "bytes" THEN "wire" \in bytesEnum ELSE "raw" \in textEnum}
                        ELSE WireMatches("wire")
          /\ phase' = "emit"
          /\ UNCHANGED <<bytesEnum,textEnum,output>>
Emit == /\ phase = "emit"
        /\ LET candidate == IF Mode = "checked" \/ matches = {"bytes"} THEN "wire" ELSE "raw"
           IN output' = IF Cardinality(WireMatches(candidate)) = 1 THEN candidate ELSE "none"
        /\ phase' = "done"
        /\ UNCHANGED <<bytesEnum,textEnum,matches>>
Spec == Init /\ [][Select \/ Emit]_vars
PreservesBytes == phase = "done" => output \in {"none","wire"}
PreservesUniqueExample == phase = "done" /\ Cardinality(WireMatches("wire")) = 1 => output = "wire"
=============================================================================
