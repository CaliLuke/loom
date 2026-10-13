-------------------- MODULE MetadataBindings --------------------
EXTENDS Naturals, FiniteSets
CONSTANT Policy
VARIABLES fields, count, admitted, encodeKeys, decodeKeys, requiredKeys, resolved
vars == <<fields, count, admitted, encodeKeys, decodeKeys, requiredKeys, resolved>>
Names == {"raw", "raw-bin", "RAW-BIN"}
Channels == {"request", "header"}
Field == [name: Names, binary: BOOLEAN, channel: Channels, implicit: BOOLEAN]
Canonical(name) == IF name = "RAW-BIN" THEN "raw-bin" ELSE name
BinaryKey(name) == Canonical(name) = "raw-bin"
Derived(f) == IF f.binary THEN "raw-bin" ELSE Canonical(f.name)
Raw == [i \in 1..count |-> fields[i].name]
Wire == [i \in 1..count |-> IF Policy = "current" THEN fields[i].name ELSE Derived(fields[i])]
Unique(keys) == \A i,j \in DOMAIN keys:
                 (i # j /\ fields[i].channel = fields[j].channel) =>
                 Canonical(keys[i]) # Canonical(keys[j])
AuthoredUnique == \A i,j \in 1..count:
                    (i # j /\ fields[i].channel = fields[j].channel) =>
                    fields[i].name # fields[j].name
ExplicitUnique(keys) == \A i,j \in DOMAIN keys:
                         (i # j /\ fields[i].channel = fields[j].channel /\
                          ~fields[i].implicit /\ ~fields[j].implicit) =>
                         Canonical(keys[i]) # Canonical(keys[j])
Init == /\ count \in {1, 2}
        /\ fields \in [1..count -> Field]
        /\ AuthoredUnique
        /\ admitted = FALSE
        /\ encodeKeys = [i \in 1..count |-> ""]
        /\ decodeKeys = encodeKeys
        /\ requiredKeys = encodeKeys
        /\ resolved = FALSE
Resolve == /\ ~resolved
           /\ admitted' = CASE Policy = "derived-owned" -> Unique(Wire)
                               [] Policy = "auth-late" -> ExplicitUnique(Wire)
                               [] Policy = "explicit-owned" -> Unique(Raw) /\
                                  (\A i \in 1..count: fields[i].binary => BinaryKey(fields[i].name))
                               [] OTHER -> TRUE
           /\ encodeKeys' = IF Policy = "explicit-owned" THEN Raw ELSE Wire
           /\ decodeKeys' = IF Policy \in {"encode-only", "explicit-owned"} THEN Raw ELSE Wire
           /\ requiredKeys' = decodeKeys'
           /\ resolved' = TRUE
           /\ UNCHANGED <<fields,count>>
Next == Resolve \/ (resolved /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
BinaryFraming == resolved /\ admitted => \A i \in 1..count: fields[i].binary => BinaryKey(encodeKeys[i])
ConsumerAgreement == resolved /\ admitted => encodeKeys = decodeKeys /\ encodeKeys = requiredKeys
DistinctBindings == resolved /\ admitted => Unique(encodeKeys)
NoOverRejection == resolved /\ Policy = "derived-owned" /\ Unique(Wire) => admitted
=============================================================================
