-------------------- MODULE DefaultRepresentation --------------------
EXTENDS Naturals, FiniteSets, TLC
CONSTANT Mode
VARIABLE sample, emitted, rejected, phase
vars == <<sample, emitted, rejected, phase>>

\* A finite flattened tree retains each value's path, physical target carrier,
\* and source carrier. Every node is independently checked after rendering.
Node(path, source, target, value) ==
    [path |-> path, source |-> source, target |-> target, value |-> value]
Cases == {
  {Node("value", "map", "object-pointer", 1),
   Node("value.x", "scalar", "primitive-pointer", 0)},
  {Node("value", "map", "named-object-pointer", 1),
   Node("value.wire_count", "scalar", "optional-Count", 0)},
  {Node("value", "array", "array", 1),
   Node("value[0]", "map", "named-object-pointer", 1),
   Node("value[0].x", "scalar", "primitive-pointer", 0)},
  {Node("value", "map", "typed-map", 1),
   Node("value[k]", "map", "nullable-object", 1),
   Node("value[k].x", "scalar", "nullable-scalar", 0)},
  {Node("value", "map", "object-pointer", 1),
   Node("value.child", "map", "object-pointer", 1),
   Node("value.child.x", "scalar", "primitive-pointer", 0)},
  {Node("value", "map", "object-pointer", 1),
   Node("value.extra", "scalar", "no-slot", 1)},
  {Node("value", "opaque", "custom-codec", 1)}
}
Supported(s) == \A n \in s : n.target \notin {"no-slot", "custom-codec"}
OuterObject(n) == n.path = "value" /\ n.target \in
    {"object-pointer", "named-object-pointer", "nullable-object"}
Legacy(n) == IF n.target \in {"array", "typed-map"} THEN n.target ELSE n.source
Emit(n) == CASE Mode = "legacy" -> Legacy(n)
             [] Mode = "shallow" -> IF OuterObject(n) THEN n.target ELSE Legacy(n)
             [] OTHER -> n.target
Init == /\ sample = {} /\ emitted = {} /\ rejected = FALSE /\ phase = "choose"
Choose == /\ phase = "choose" /\ sample' \in Cases /\ phase' = "render"
          /\ UNCHANGED <<emitted, rejected>>
Render == /\ phase = "render"
          /\ rejected' = ~Supported(sample)
          /\ emitted' = IF Supported(sample)
                THEN {[path |-> n.path, carrier |-> Emit(n), value |-> n.value] : n \in sample}
                ELSE {}
          /\ phase' = "done" /\ UNCHANGED sample
Next == Choose \/ Render
Spec == Init /\ [][Next]_vars
Typed == phase = "done" /\ ~rejected =>
    \A n \in sample : \E e \in emitted : e.path = n.path /\ e.carrier = n.target
PreservesValues == phase = "done" /\ ~rejected =>
    \A n \in sample : \E e \in emitted : e.path = n.path /\ e.value = n.value
NoSilentLoss == phase = "done" => rejected = ~Supported(sample)
=============================================================================
