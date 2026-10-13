-------------------- MODULE ConstructorOwnership --------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
VARIABLES pending, bindings, definitions, bodyTypes
vars == <<pending, bindings, definitions, bodyTypes>>

Ids == {"handlerPayload", "responsePayload", "responseBody", "required",
        "optional", "requiredCopy", "left", "right"}
Target(i) == CASE i = "handlerPayload" -> "Handler"
              [] i = "responsePayload" -> "ServiceResponse"
              [] i = "responseBody" -> "WireResponse"
              [] OTHER -> "Text"
Argument(i) == CASE i \in {"required", "requiredCopy"} -> "RequiredView"
                [] i = "optional" -> "OptionalView"
                [] i \in {"left", "right"} -> "PairView"
                [] OTHER -> i
Behavior(i) == CASE i \in {"required", "requiredCopy", "optional"} -> "value"
                [] OTHER -> i
Identity(i) == <<Target(i), Argument(i), Behavior(i)>>
Signature(i) == <<Target(i), Argument(i)>>
Preferred(i) == CASE i = "handlerPayload" -> 1
                 [] i \in {"responsePayload", "responseBody"} -> 2
                 [] OTHER -> 3
Fixed == {<<1, 0>>, <<3, 1>>}
NoBinding == <<0, 0>>
Names == {d.name : d \in definitions} \cup Fixed
Fresh(i) == <<Preferred(i), CHOOSE suffix \in 0..8 :
              /\ <<Preferred(i), suffix>> \notin Names
              /\ \A earlier \in 0..(suffix-1) : <<Preferred(i), earlier>> \in Names>>
Key(i) == IF Mode = "signature" THEN Signature(i) ELSE Identity(i)
Prior(i) == {j \in Ids \ pending : Key(j) = Key(i)}
Chosen(i) == IF Prior(i) # {} THEN bindings[CHOOSE j \in Prior(i) : TRUE]
             ELSE IF Mode \in {"current", "identity"} THEN <<Preferred(i), 0>>
             ELSE Fresh(i)

Init == /\ pending = Ids
        /\ bindings = [i \in Ids |-> NoBinding]
        /\ definitions = {}
        /\ bodyTypes = {}
Step(i) == /\ i \in pending
           /\ LET name == Chosen(i)
                  suppressed == Mode \in {"current", "lexical"} /\ Target(i) \in bodyTypes
              IN /\ bindings' = [bindings EXCEPT ![i] = name]
                 /\ definitions' = IF suppressed THEN definitions
                     ELSE definitions \cup {[name |-> name, identity |-> Identity(i)]}
           /\ bodyTypes' = bodyTypes \cup {Target(i)}
           /\ pending' = pending \ {i}
Next == \E i \in Ids : Step(i)
Spec == Init /\ [][Next]_vars

Unique == /\ \A d \in definitions : d.name \notin Fixed
          /\ \A a,b \in definitions : a.name = b.name => a.identity = b.identity
Resolved == \A i \in Ids \ pending :
              \E d \in definitions : d.name = bindings[i] /\ d.identity = Identity(i)
=====================================================================
