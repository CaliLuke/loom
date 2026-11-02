--------------------------- MODULE PresenceDecode ---------------------------
EXTENDS Naturals, Sequences
CONSTANTS Mode, MaxDepth
VARIABLES chain, position, options
vars == <<chain, position, options>>
Caller == {"strict", "custom"}
Defaults == {"booleanKeys"}
Init == /\ chain \in UNION {[1..n -> {"optional", "nullable"}]: n \in 1..MaxDepth}
        /\ position = 1
        /\ options = Caller
Decode == /\ position <= Len(chain)
          /\ options' = CASE Mode = "restart" -> Defaults
                           [] Mode = "inherit" -> options
                           [] OTHER -> options \cup Defaults
          /\ position' = position + 1
          /\ UNCHANGED chain
Spec == Init /\ [][Decode]_vars
CallerPreserved == Caller \subseteq options
BooleanKeysAvailable == position > 1 => "booleanKeys" \in options
=============================================================================
