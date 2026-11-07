---- MODULE IntegerTextBoundary ----
\* Value-level representability at an HTTP integer text boundary.
\* Storage and wire domains are abstract integer intervals; Go conversion is
\* modular wrapping into the destination interval. Mode selects the emitter.
EXTENDS Integers

CONSTANTS Storages, Wires, Values, Mode

VARIABLES storage, wire, value, phase, published, text, decodeState, decoded

Dom(d) == d.lo..d.hi
Wrap(v, d) == d.lo + ((v - d.lo) % (d.hi - d.lo + 1))
Checked(v, from, to) == Wrap(v, to) \in Dom(from) /\ Wrap(Wrap(v, to), from) = v /\ ((v < 0) = (Wrap(v, to) < 0))

vars == <<storage, wire, value, phase, published, text, decodeState, decoded>>

Init ==
  /\ storage \in Storages /\ wire \in Wires
  /\ value \in Values \cap Dom(storage)
  /\ phase = "encode" /\ published = FALSE /\ text = 0 /\ decodeState = "none" /\ decoded = 0

\* Encode: "unchecked" models the parent cast to the wire carrier; "checked"
\* models the candidate round-trip/sign check before publication.
Encode ==
  /\ phase = "encode"
  /\ IF Mode = "unchecked" \/ Checked(value, storage, wire)
       THEN /\ published' = TRUE /\ text' = Wrap(value, wire) /\ phase' = "decode"
       ELSE /\ published' = FALSE /\ text' = 0 /\ phase' = "done"
  /\ UNCHANGED <<storage, wire, value, decodeState, decoded>>

\* Decode: parse in the wire domain (the existing parser), then convert to storage.
Decode ==
  /\ phase = "decode"
  /\ IF text \in Dom(wire) /\ (Mode = "unchecked" \/ Checked(text, wire, storage))
       THEN decodeState' = "accepted" /\ decoded' = Wrap(text, storage)
       ELSE decodeState' = "rejected" /\ decoded' = 0
  /\ phase' = "done"
  /\ UNCHANGED <<storage, wire, value, published, text>>

Next == Encode \/ Decode \/ (phase = "done" /\ UNCHANGED vars)

Spec == Init /\ [][Next]_vars

\* Published text always denotes the stored value and lies in the wire domain.
FaithfulPublication == published => (text = value /\ text \in Dom(wire))
\* A value outside the joint domain is never published.
ErrorBeforePublication == (value \notin Dom(wire)) => ~published
\* Decoding never changes a published value; accepted values round-trip.
DecodePreservesValue == (decodeState = "accepted") => decoded = text
\* Every representable value is accepted end to end (no over-rejection).
RepresentableAccepted == (phase = "done" /\ value \in Dom(wire)) => (decodeState = "accepted" /\ decoded = value)
====
