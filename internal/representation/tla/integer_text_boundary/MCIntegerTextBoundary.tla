---- MODULE MCIntegerTextBoundary ----
\* Scaled domains: narrow signed (int8), unsigned (uint8) and wider signed
\* storage against signed and unsigned wires. Opaque storage is the same
\* interval semantics once the emitted ^T(0) assertion proves integer storage.
EXTENDS IntegerTextBoundary
MCStorages == {[lo |-> -128, hi |-> 127], [lo |-> 0, hi |-> 255], [lo |-> -512, hi |-> 511]}
MCWires == {[lo |-> -256, hi |-> 255], [lo |-> 0, hi |-> 511]}
MCValues == {-512, -300, -129, -128, -1, 0, 1, 127, 128, 200, 255, 256, 300, 511}
====
