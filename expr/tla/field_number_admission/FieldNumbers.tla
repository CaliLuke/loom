---------------------------- MODULE FieldNumbers ----------------------------
EXTENDS Integers, Sequences
CONSTANT Strategy
VARIABLES authored, selected, emitted, phase, accepted, published
vars == <<authored, selected, emitted, phase, accepted, published>>

\* Bounds are reduced, but preserve reserved gaps and machine wrap ordering.
Field(owner, raw, explicit, seed, offset, included) ==
 [owner |-> owner, raw |-> raw, explicit |-> explicit,
  seed |-> seed, offset |-> offset, included |-> included]
F(owner, raw, number) == Field(owner, raw, number, 0, 0, TRUE)
Designs == {
 <<F("a", "0", 0)>>,
 <<F("a", "negative", -1)>>,
 <<F("a", "nonnumeric", -2)>>,
 <<F("a", "reserved", 4)>>,
 <<F("a", "too-large", 8)>>,
 <<F("a", "01", 1), F("a", "1", 1)>>,
 <<F("a", "1", 1), F("b", "1", 1)>>,
 <<F("a", "1", 1), F("a", "7", 7)>>,
 <<Field("a", "derived3", -99, 3, 0, TRUE),
   Field("a", "derived4", -99, 3, 1, TRUE)>>,
 <<Field("a", "derived7", -99, 7, 0, TRUE),
   Field("a", "derived8", -99, 7, 1, TRUE)>>,
 <<F("a", "2", 2), F("a", "3", 3),
   Field("a", "wrap", -99, 15, 2, TRUE)>>,
 <<Field("a", "override", 1, 5, 0, TRUE),
   Field("a", "derived6", -99, 5, 1, TRUE)>>,
 <<Field("a", "override", 1, -2, 0, TRUE),
   Field("a", "override2", 2, -2, 1, TRUE)>>,
 <<Field("a", "metadata", 4, 0, 0, FALSE), F("a", "1", 1)>>
}
Exact(f) == IF f.explicit = -99 THEN f.seed + f.offset ELSE f.explicit
Number(f) == IF Strategy \in {"current", "bounds-only", "wrapped"}
              /\ f.explicit = -99 THEN Exact(f) % 16 ELSE Exact(f)
Legal(n) == n \in 1..7 /\ n \notin 4..5
Selected(d) == SelectSeq(d, LAMBDA f : f.included)
LegalFields(d) == \A i \in DOMAIN d : Legal(Exact(d[i]))
Unique(d, global) == \A i,j \in DOMAIN d :
 i # j /\ (global \/ d[i].owner = d[j].owner) => Exact(d[i]) # Exact(d[j])
RawUnique(d) == \A i,j \in DOMAIN d :
 i # j /\ d[i].owner = d[j].owner => d[i].raw # d[j].raw
Valid(d) == LegalFields(d) /\ Unique(d, FALSE)
Admit(d) == CASE Strategy = "current" -> RawUnique(d)
 [] Strategy = "bounds-only" -> LegalFields(d) /\ RawUnique(d)
 [] Strategy = "global" -> LegalFields(d) /\ Unique(d, TRUE)
 [] Strategy = "early" -> Valid(authored)
 [] OTHER -> Valid(d)

Init == /\ authored \in Designs
        /\ selected = <<>> /\ emitted = <<>>
        /\ phase = "authored" /\ accepted = FALSE /\ published = FALSE
Select == /\ phase = "authored"
          /\ selected' = Selected(authored) /\ phase' = "selected"
          /\ UNCHANGED <<authored, emitted, accepted, published>>
Derive == /\ phase = "selected"
          /\ emitted' = [i \in DOMAIN selected |->
                          [selected[i] EXCEPT !.explicit = Number(selected[i])]]
          /\ phase' = "derived"
          /\ UNCHANGED <<authored, selected, accepted, published>>
Validate == /\ phase = "derived"
            /\ accepted' = Admit(emitted) /\ phase' = "validated"
            /\ UNCHANGED <<authored, selected, emitted, published>>
Publish == /\ phase = "validated"
           /\ published' = (accepted /\ (Strategy # "container" \/
                \A i \in DOMAIN selected : selected[i].seed >= 0))
           /\ phase' = "published"
           /\ UNCHANGED <<authored, selected, emitted, accepted>>
Next == Select \/ Derive \/ Validate \/ Publish
Spec == Init /\ [][Next]_vars
AdmissionSafe == phase \in {"validated", "published"} /\ accepted => Valid(Selected(authored))
ValidReuse == phase \in {"validated", "published"} /\ Valid(Selected(authored)) => accepted
NoNumberRewrite == phase \in {"validated", "published"} /\ accepted =>
 \A i \in DOMAIN emitted : Exact(emitted[i]) = Exact(selected[i])
PublishedWhenAccepted == phase = "published" /\ accepted => published
=============================================================================
