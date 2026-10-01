---------------------- MODULE PreparedExampleRoles ----------------------
EXTENDS Naturals, Sequences
CONSTANT Mode
VARIABLES consumer, owner, structural, example, frames, steps, wireCodec, exampleCodec
vars == <<consumer, owner, structural, example, frames, steps, wireCodec, exampleCodec>>
Neutral == <<>>
Structure(path) == IF consumer = "standalone" THEN Neutral ELSE path
Init ==
    /\ consumer \in {"standalone", "transport"}
    /\ wireCodec \in {"json", "text", "raw", "form", "multipart"}
    /\ exampleCodec = IF Mode = "inherited-wire" THEN wireCodec ELSE "json"
    /\ owner = <<0>>
    /\ structural = IF Mode = "conflated" THEN <<0>> ELSE Structure(<<0>>)
    /\ example = IF Mode = "clear-both" /\ consumer = "standalone"
                  THEN Neutral ELSE <<0>>
    /\ frames = <<>>
    /\ steps = 0
Enter ==
    /\ steps < 8 /\ Len(owner) < 3
    /\ \E edge \in {1, 2}:
        LET child == Append(owner, edge)
        IN /\ owner' = child
           /\ frames' = Append(frames, <<structural, example>>)
           /\ structural' = IF Mode = "conflated" THEN child ELSE Structure(child)
           /\ example' = IF Mode = "stale-child" THEN example
                          ELSE IF Mode = "clear-both" /\ consumer = "standalone"
                               THEN Neutral ELSE child
    /\ steps' = steps + 1
    /\ UNCHANGED <<consumer, wireCodec, exampleCodec>>
\* Leaving a scope, including unwinding it after an exception, restores the pair.
Leave ==
    /\ steps < 8 /\ Len(owner) > 1
    /\ owner' = SubSeq(owner, 1, Len(owner) - 1)
    /\ structural' = frames[Len(frames)][1]
    /\ example' = IF Mode = "leaky-restore" THEN example
                  ELSE frames[Len(frames)][2]
    /\ frames' = SubSeq(frames, 1, Len(frames) - 1)
    /\ steps' = steps + 1
    /\ UNCHANGED <<consumer, wireCodec, exampleCodec>>
Spec == Init /\ [][Enter \/ Leave]_vars
StructuralAuthority == structural = Structure(owner)
ExampleAuthority == example = owner
\* The document observes a JSON value; the runtime transport keeps its codec.
\* Custom codecs and node-local SSE policies are outside this finite root model.
ExampleEncodingAuthority == exampleCodec = "json"
=============================================================================
