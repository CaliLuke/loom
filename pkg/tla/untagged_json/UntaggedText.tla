------------------------- MODULE UntaggedText -------------------------
\* Branch-indexed text union matching for #1014. Each branch owns a text
\* codec resolved by the shared parameter codec owner (#968). Matching keeps
\* two accounting domains, as MatchUntaggedJSON does: the projected schema
\* (width independent) and the typed carrier (decode plus validation). The
\* server accepts only one schema match and one carrier match with the same
\* identity, and assigns only then. The client encodes the selected branch and
\* runs the same matcher over its own text before publishing.
EXTENDS Naturals, FiniteSets, Sequences
CONSTANTS Profile, ClientMode, ServerMode

\* Profiles fix ordered branch lists; TLC configurations cannot spell tuples.
Branches == CASE Profile = "pattern" -> <<"Lower", "Int", "IntList">>
              [] Profile = "widths" -> <<"Int", "Int32", "Lower">>
              [] Profile = "reversed" -> <<"IntList", "Int", "Str">>
              [] OTHER -> <<"Str", "Int", "IntList">>

\* "big" stands for 2147483648: an integer that int64 holds and int32 does not.
\* "?" is decoded by no constrained branch.
Texts == {"a", "5", "6", "5,6", "", "big", "?"}
N == Len(Branches)
Idx == 1..N
Sentinel == <<0, "old">>

\* Values are canonical string tokens so TLC compares one value kind. Each
\* codec is injective on its domain; Decode inverts Encode (#968 obligation).
Dom(k) == CASE k = "Str" -> Texts
            [] k = "Lower" -> {"a"}
            [] k = "Int" -> {"5", "6", "big"}
            [] k = "Int32" -> {"5", "6"}
            [] k = "IntList" -> {"[]", "[5]", "[6]", "[5,6]"}
Enc(k, v) == CASE k \in {"Str", "Lower", "Int", "Int32"} -> v
               [] k = "IntList" -> CASE v = "[]" -> ""
                                     [] v = "[5]" -> "5"
                                     [] v = "[6]" -> "6"
                                     [] OTHER -> "5,6"
\* Schema domain: the branch codec parses the text into the JSON data model
\* and the projected schema validates it. Integer type has no width.
SchemaOK(k, w) == CASE k = "Str" -> TRUE
                    [] k = "Lower" -> w = "a"
                    [] k \in {"Int", "Int32"} -> w \in {"5", "6", "big"}
                    [] k = "IntList" -> w \in {"", "5", "6", "5,6"}
\* Carrier domain: typed decode into the Go carrier, then validation.
CarrierOK(k, w) == SchemaOK(k, w) /\ ~(k = "Int32" /\ w = "big")
TextVal(k, w) == CASE k \in {"Str", "Lower", "Int", "Int32"} -> w
                   [] k = "IntList" -> CASE w = "" -> "[]"
                                         [] w = "5" -> "[5]"
                                         [] w = "6" -> "[6]"
                                         [] OTHER -> "[5,6]"
\* Rejected candidate: coerce the text into JSON first and match JSON kinds.
CoerceOK(k, w) == CASE k = "Str" -> w \notin {"5", "6", "big"}
                    [] k = "Lower" -> w = "a"
                    [] k = "Int" -> w \in {"5", "6", "big"}
                    [] k = "Int32" -> w \in {"5", "6"}
                    [] k = "IntList" -> FALSE
SSchema(k, w) == IF ServerMode = "coerce" THEN CoerceOK(k, w) ELSE SchemaOK(k, w)
SCarrier(k, w) == IF ServerMode = "coerce" THEN CoerceOK(k, w) ELSE CarrierOK(k, w)

ASSUME \A j \in Idx : \A v \in Dom(Branches[j]) :
         /\ CarrierOK(Branches[j], Enc(Branches[j], v))
         /\ TextVal(Branches[j], Enc(Branches[j], v)) = v

SchemaOracle(w) == {j \in Idx : SchemaOK(Branches[j], w)}
CarrierOracle(w) == {j \in Idx : CarrierOK(Branches[j], w)}
Unique(w) == Cardinality(SchemaOracle(w)) = 1 /\ SchemaOracle(w) = CarrierOracle(w)

VARIABLES phase, sel, val, text, sent, ci, csc, csl, cdc, cdl,
          si, ssc, ssl, sdc, sdl, sval, dest, ran, accepted
vars == <<phase, sel, val, text, sent, ci, csc, csl, cdc, cdl,
          si, ssc, ssl, sdc, sdl, sval, dest, ran, accepted>>

Init == /\ phase = "choose"
        /\ sel = 0 /\ val = "none" /\ text = ""
        /\ sent = FALSE /\ ci = 1 /\ csc = 0 /\ csl = 0 /\ cdc = 0 /\ cdl = 0
        /\ si = 1 /\ ssc = 0 /\ ssl = 0 /\ sdc = 0 /\ sdl = 0 /\ sval = "none"
        /\ dest = Sentinel /\ ran = FALSE /\ accepted = FALSE

\* A generated client starts from a typed selected branch.
Choose == /\ phase = "choose"
          /\ \E j \in Idx : \E v \in Dom(Branches[j]) :
               /\ sel' = j /\ val' = v
               /\ text' = Enc(Branches[j], v)
          /\ phase' = "check"
          /\ UNCHANGED <<sent, ci, csc, csl, cdc, cdl, si, ssc, ssl, sdc, sdl, sval,
                         dest, ran, accepted>>

\* An arbitrary peer may send any text; only the server runs.
Inject == /\ phase = "choose"
          /\ \E w \in Texts : text' = w
          /\ phase' = "decode"
          /\ UNCHANGED <<sel, val, sent, ci, csc, csl, cdc, cdl, si, ssc, ssl, sdc, sdl, sval,
                         dest, ran, accepted>>

\* The client's re-decode check uses the same two accounting domains.
CheckStep == /\ phase = "check" /\ ci <= N
             /\ IF SchemaOK(Branches[ci], text)
                   THEN csc' = csc + 1 /\ csl' = ci ELSE UNCHANGED <<csc, csl>>
             /\ IF CarrierOK(Branches[ci], text)
                   THEN cdc' = cdc + 1 /\ cdl' = ci ELSE UNCHANGED <<cdc, cdl>>
             /\ ci' = ci + 1
             /\ UNCHANGED <<phase, sel, val, text, sent, si, ssc, ssl, sdc, sdl, sval,
                            dest, ran, accepted>>

CheckDone == /\ phase = "check" /\ ci > N
             /\ IF ClientMode = "nocheck" \/ (csc = 1 /\ cdc = 1 /\ csl = sel /\ cdl = sel)
                   THEN sent' = TRUE /\ phase' = "decode"
                   ELSE sent' = FALSE /\ phase' = "done"
             /\ UNCHANGED <<sel, val, text, ci, csc, csl, cdc, cdl, si, ssc, ssl, sdc, sdl,
                            sval, dest, ran, accepted>>

DecodeStep == /\ phase = "decode" /\ si <= N
              /\ ran' = TRUE
              /\ IF ServerMode = "first" /\ sdc >= 1
                    THEN /\ si' = N + 1
                         /\ UNCHANGED <<ssc, ssl, sdc, sdl, sval, dest>>
                    ELSE /\ si' = si + 1
                         /\ IF SSchema(Branches[si], text)
                               THEN ssc' = ssc + 1 /\ ssl' = si ELSE UNCHANGED <<ssc, ssl>>
                         /\ IF SCarrier(Branches[si], text)
                               THEN /\ sdc' = sdc + 1 /\ sdl' = si
                                    /\ sval' = TextVal(Branches[si], text)
                                    /\ dest' = IF ServerMode = "eager"
                                                  THEN <<si, TextVal(Branches[si], text)>>
                                                  ELSE dest
                               ELSE UNCHANGED <<sdc, sdl, sval, dest>>
              /\ UNCHANGED <<phase, sel, val, text, sent, ci, csc, csl, cdc, cdl, accepted>>

Accept == CASE ServerMode = "carrier-only" -> sdc = 1
            [] ServerMode = "first" -> sdc = 1
            [] OTHER -> ssc = 1 /\ sdc = 1 /\ ssl = sdl

DecodeDone == /\ phase = "decode" /\ si > N
              /\ ran' = TRUE
              /\ accepted' = Accept
              /\ dest' = IF Accept THEN <<sdl, sval>> ELSE dest
              /\ phase' = "done"
              /\ UNCHANGED <<sel, val, text, sent, ci, csc, csl, cdc, cdl, si, ssc, ssl, sdc, sdl, sval>>

Next == Choose \/ Inject \/ CheckStep \/ CheckDone \/ DecodeStep \/ DecodeDone
Spec == Init /\ [][Next]_vars

\* Acceptance means exactly one schema match and one carrier match, the same.
ServerSound == phase = "done" /\ accepted =>
                 /\ Unique(text)
                 /\ dest[1] \in SchemaOracle(text)
                 /\ dest[2] = TextVal(Branches[dest[1]], text)
\* A unique, agreeing match is never rejected.
ServerComplete == phase = "done" /\ ran /\ Unique(text) => accepted
\* A sent value is received as the same branch and value.
Agreement == phase = "done" /\ sent => accepted /\ dest = <<sel, val>>
\* The client never publishes text that is not uniquely attributable.
NoAmbiguousSend == sent => Unique(text)
\* The caller's destination changes only with acceptance, in every state.
Transactional == ~accepted => dest = Sentinel
\* No-match input is rejected with the destination untouched.
ZeroMatchRejected == phase = "done" /\ ran /\ SchemaOracle(text) = {} =>
                       ~accepted /\ dest = Sentinel

\* Reachability witnesses, checked as expected violations.
NoZeroMatchWitness == ~(phase = "done" /\ ran /\ SchemaOracle(text) = {})
NoWidthSplitWitness == ~(phase = "done" /\ ran /\ SchemaOracle(text) # CarrierOracle(text))
=============================================================================
