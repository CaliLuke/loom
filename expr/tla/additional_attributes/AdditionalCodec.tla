---- MODULE AdditionalCodec ----
\* Generated codec for Probe: optional declared id:Int, non-nullable Int extras.
\* Carrier selects how extras decode; CollisionCheck guards outbound encoding.
EXTENDS TLC

CONSTANTS Carrier, CollisionCheck

WireValues == {"zero", "neg", "str", "null"}
Absent == "absent"
Int == {"zero", "neg"}
Keys == {"id", "x"}

\* Wire documents a decoder can receive.
Wires == [Keys -> WireValues \cup {Absent}]
Spec(w) == \A k \in Keys : w[k] # Absent => w[k] \in Int

\* Declared fields already use Loom's presence-preserving decode types.
ExtraDecodes(v) == ~(Carrier = "optional" /\ v = "null")
ExtraValue(v) == IF Carrier = "plain" /\ v = "null" THEN "zero" ELSE v
Generated(w) ==
    /\ w["id"] # Absent => w["id"] \in Int
    /\ w["x"] # Absent => ExtraDecodes(w["x"]) /\ ExtraValue(w["x"]) \in Int

\* Service values constructed by applications before outbound encoding.
ServiceValues == [declared : Int \cup {Absent}, extras : UNION {[D -> Int] : D \in SUBSET Keys}]
EncodeOK(sv) ==
    /\ ~(CollisionCheck /\ "id" \in DOMAIN sv.extras)
    /\ ~(sv.declared # Absent /\ "id" \in DOMAIN sv.extras)   \* json/v2 duplicate-name error
Wire(sv) == [k \in Keys |->
    IF k = "id" THEN (IF sv.declared # Absent THEN sv.declared
                      ELSE IF "id" \in DOMAIN sv.extras THEN sv.extras["id"] ELSE Absent)
    ELSE IF k \in DOMAIN sv.extras THEN sv.extras[k] ELSE Absent]
Decode(w) == [declared |-> w["id"],
              extras |-> [k \in {j \in {"x"} : w[j] # Absent} |-> w[k]]]

VARIABLE done
Init == done = FALSE
Next == UNCHANGED done

DecodeFaithful == \A w \in Wires : Generated(w) = Spec(w)
RoundTrip == \A sv \in ServiceValues : EncodeOK(sv) => Decode(Wire(sv)) = sv
====
