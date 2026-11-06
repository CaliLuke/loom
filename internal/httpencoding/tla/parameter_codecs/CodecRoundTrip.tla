------------------------- MODULE CodecRoundTrip -------------------------
\* #968 wire codecs: encoder admission, transport transformations and decoders
\* for path, query, header and cookie styles over a delimiter-rich alphabet.
EXTENDS Naturals, Sequences, FiniteSets
CONSTANT Policy
VARIABLES codec, value, required, normalize, hexcase, peer, phase, admitted, wire, decoded
vars == <<codec, value, required, normalize, hexcase, peer, phase, admitted, wire, decoded>>

\* "e" stands for a non-ASCII character such as U+00E9 (one UTF-8 escape token).
\* "P" stands for the literal three-character text "%2E" (escaped as "%252E").
D == {"a", ",", ".", ";", "=", " ", "|", "e", "P"}
Str == {<<>>} \cup {<<c>> : c \in D}
Lists == {<<>>} \cup {<<s>> : s \in Str} \cup {<<s, t>> : s, t \in Str}
Gone == <<"#">>
Objs == [{"p", "q"} -> Str \cup {Gone}]
NoObj == [k \in {"p", "q"} |-> Gone]
\* Scalar-valued maps carry at most one dynamic entry <<key, value>>.
MapVals == {<<>>} \cup {<< <<k, v>> >> : k, v \in Str}
Absent == [shape |-> "absent", s |-> <<>>, l |-> <<>>, o |-> NoObj, m |-> <<>>]
Reject == [shape |-> "reject", s |-> <<>>, l |-> <<>>, o |-> NoObj, m |-> <<>>]
Sc(s) == [shape |-> "scalar", s |-> s, l |-> <<>>, o |-> NoObj, m |-> <<>>]
Li(l) == [shape |-> "sequence", s |-> <<>>, l |-> l, o |-> NoObj, m |-> <<>>]
Ob(o) == [shape |-> "object", s |-> <<>>, l |-> <<>>, o |-> o, m |-> <<>>]
Mp(m) == [shape |-> "map", s |-> <<>>, l |-> <<>>, o |-> NoObj, m |-> m]

C(l, s, e) == [loc |-> l, style |-> s, explode |-> e]
Codecs == {C("path", s, e) : s \in {"simple", "matrix", "label"}, e \in BOOLEAN}
          \cup {C("query", "form", e) : e \in BOOLEAN}
          \cup {C("query", s, FALSE) : s \in {"spaceDelimited", "pipeDelimited", "deepObject"}}
          \cup {C("header", "simple", e) : e \in BOOLEAN}
          \cup {C("cookie", "cookie", e) : e \in BOOLEAN}
          \cup {C("cookie", "form", FALSE)}
Shapes(c) == CASE c.style \in {"spaceDelimited", "pipeDelimited"} -> {"sequence", "object", "map"}
              [] c.style = "deepObject" -> {"object", "map"}
              [] OTHER -> {"scalar", "sequence", "object", "map"}
Values(c) == UNION {CASE sh = "scalar" -> {Sc(s) : s \in Str}
                      [] sh = "sequence" -> {Li(l) : l \in Lists}
                      [] sh = "map" -> {Mp(m) : m \in MapVals}
                      [] OTHER -> {Ob(o) : o \in Objs} : sh \in Shapes(c)}

\* ---- token helpers ----
Esc(c) == CASE c = "," -> "%2C" [] c = "." -> "%2E" [] c = ";" -> "%3B"
            [] c = "=" -> "%3D" [] c = " " -> "%20" [] c = "|" -> "%7C" [] c = "e" -> "%C3%A9"
            [] c = "P" -> "%252E" [] OTHER -> c
Unesc(t) == CASE t = "%2C" -> "," [] t = "%2E" -> "." [] t = "%2e" -> "." [] t = "%3B" -> ";" [] t = "%3D" -> "="
              [] t = "%20" -> " " [] t = "%7C" -> "|" [] t = "%5B" -> "[" [] t = "%5D" -> "]"
              [] t = "+" -> " " [] t = "%C3%A9" -> "e" [] t = "%252E" -> "P" [] OTHER -> t
Unreserved(c) == c \in {"a", "."}
EP(s) == [i \in 1..Len(s) |-> IF Unreserved(s[i]) THEN s[i] ELSE Esc(s[i])]
EQ(s) == [i \in 1..Len(s) |-> IF Unreserved(s[i]) THEN s[i] ELSE IF s[i] = " " THEN "+" ELSE Esc(s[i])]
\* Rejected candidate: escape dots in label items as %2E.
EL(s) == IF Policy = "label-pct-dot" THEN [i \in 1..Len(s) |-> IF s[i] = "." THEN "%2E" ELSE EP(s)[i]] ELSE EP(s)
UnescAll(t) == [i \in 1..Len(t) |-> Unesc(t[i])]
Map(f(_), l) == [i \in 1..Len(l) |-> f(l[i])]
RECURSIVE Join(_, _)
Join(items, d) == IF items = <<>> THEN <<>>
                  ELSE IF Len(items) = 1 THEN Head(items)
                  ELSE Head(items) \o d \o Join(Tail(items), d)
RECURSIVE SplitOn(_, _)
SplitOn(t, d) == IF t = <<>> THEN << <<>> >>
                 ELSE LET rest == SplitOn(Tail(t), d)
                      IN IF Head(t) = d THEN << <<>> >> \o rest
                         ELSE << <<Head(t)>> \o Head(rest) >> \o Tail(rest)
Pos(t, x) == IF \E i \in 1..Len(t) : t[i] = x THEN CHOOSE i \in 1..Len(t) : t[i] = x /\ \A j \in 1..(i - 1) : t[j] # x ELSE 0
Before(t, x) == IF Pos(t, x) = 0 THEN t ELSE SubSeq(t, 1, Pos(t, x) - 1)
After(t, x) == IF Pos(t, x) = 0 THEN <<>> ELSE SubSeq(t, Pos(t, x) + 1, Len(t))
RECURSIVE TrimL(_)
TrimL(t) == IF t # <<>> /\ Head(t) = " " THEN TrimL(Tail(t)) ELSE t
RECURSIVE TrimR(_)
TrimR(t) == IF t # <<>> /\ t[Len(t)] = " " THEN TrimR(SubSeq(t, 1, Len(t) - 1)) ELSE t
Trim(t) == TrimR(TrimL(t))
Props(o) == SelectSeq(<<"p", "q">>, LAMBDA k: o[k] # Gone)
Contains(s, ch) == \E i \in 1..Len(s) : s[i] = ch
MapKeys(v) == {v.m[i][1] : i \in 1..Len(v.m)}
\* Items are the scalar texts a value spells: elements, property values, and
\* map keys and values.
Items(v) == CASE v.shape = "scalar" -> {v.s} [] v.shape = "sequence" -> {v.l[i] : i \in 1..Len(v.l)}
            [] v.shape = "map" -> MapKeys(v) \cup {v.m[i][2] : i \in 1..Len(v.m)}
            [] OTHER -> {v.o[k] : k \in {k2 \in {"p", "q"} : v.o[k2] # Gone}}
EmptyValue(v) == \/ (v.shape = "sequence" /\ v.l = <<>>) \/ (v.shape = "object" /\ Props(v.o) = <<>>)
                 \/ (v.shape = "map" /\ v.m = <<>>)
\* Entries as raw <<key tokens, value tokens>>; fixed properties use p/q names.
Entries(v) == IF v.shape = "map" THEN v.m ELSE Map(LAMBDA k: <<<<k>>, v.o[k]>>, Props(v.o))
TChar(c) == c \in {"a", ".", "|", "P"}

\* Codecs whose empty collection has no wire key (Rule A).
NoKeyEmpty(c) == \/ (c.loc = "query" /\ c.style = "form" /\ c.explode)
                 \/ (c.loc = "query" /\ c.style = "deepObject")
                 \/ (c.loc = "cookie" /\ c.style = "cookie" /\ c.explode)
\* Codecs where [] and [""] share one spelling ("n=").
SingletonCollides(c) == c.loc \in {"query", "cookie"} /\ ~(c.style \in {"form", "cookie"} /\ c.explode)
                        /\ c.style # "deepObject"
Delim(c) == CASE c.style = "spaceDelimited" -> " " [] c.style = "pipeDelimited" -> "|" [] OTHER -> ","

\* ---- encoder (design admission + spelling) ----
PathDelim(c) == IF c.style = "label" /\ c.explode THEN "." ELSE ","
PathEsc(c, s) == IF c.style = "label" THEN EL(s) ELSE EP(s)
PathKey(c, v, k) == IF v.shape = "map" THEN PathEsc(c, k) ELSE k
PathPairs(c, v) == Map(LAMBDA e: PathKey(c, v, e[1]) \o (IF c.explode THEN <<"=">> ELSE <<",">>) \o PathEsc(c, e[2]),
                       Entries(v))
PathEncode(c, v) ==
  CASE c.style = "matrix" ->
         CASE v.shape = "scalar" -> IF v.s = <<>> THEN <<";", "n">> ELSE <<";", "n", "=">> \o EP(v.s)
           [] v.shape = "sequence" ->
                IF v.l = <<>> THEN <<";", "n">>
                ELSE IF c.explode THEN Join(Map(LAMBDA s: <<";", "n", "=">> \o EP(s), v.l), <<>>)
                ELSE <<";", "n", "=">> \o Join(Map(LAMBDA s: EP(s), v.l), <<",">>)
           [] OTHER ->
                IF c.explode THEN Join(Map(LAMBDA p: <<";">> \o p, PathPairs(c, v)), <<>>)
                ELSE IF Entries(v) = <<>> THEN <<";", "n">>
                ELSE <<";", "n", "=">> \o Join(PathPairs(c, v), <<",">>)
    [] OTHER ->
         LET pre == IF c.style = "label" THEN <<".">> ELSE <<>> IN
         CASE v.shape = "scalar" -> pre \o PathEsc(c, v.s)
           [] v.shape = "sequence" -> pre \o Join(Map(LAMBDA s: PathEsc(c, s), v.l), <<PathDelim(c)>>)
           [] OTHER -> pre \o Join(PathPairs(c, v), <<PathDelim(c)>>)
\* RFC 6570 3.2.1: a list or associative array with zero members is undefined
\* and expands to nothing, not even the label prefix; a defined empty string
\* keeps the prefix (3.2.5). Matrix keeps its ";name" marker (design section 5).
PathSpell(c, v) == IF c.style # "matrix" /\ EmptyValue(v) THEN <<>> ELSE PathEncode(c, v)
DotSegment(seg) == seg \in {<<>>, <<".">>, <<".", ".">>}
\* The complete serialized segment is protected only when it is exactly "." or
\* "..": every other segment keeps its structural separators literal.
Protect(w) == CASE w = <<".">> -> <<"%2E">> [] w = <<".", ".">> -> <<"%2E", "%2E">> [] OTHER -> w
Pct2E(t) == t \in {"%2E", "%2e"}
\* RFC 3986 2.1: hex digits are case-insensitive; both spellings are protected.
Protected(t) == Len(t) \in {1, 2} /\ \A i \in 1..Len(t) : Pct2E(t[i])
Unprotect(t) == IF Protected(t) /\ (Policy # "dot-case-sensitive" \/ \A i \in 1..Len(t) : t[i] = "%2E")
                THEN [i \in 1..Len(t) |-> "."] ELSE t
\* dot-reject is the rejected candidate that refuses whole dot segments instead
\* of protecting them. An empty segment cannot address a whole-segment route.
PathAdmit(c, v, raw) ==
  /\ raw # <<>>
  /\ Policy = "dot-reject" => ~DotSegment(raw)
  /\ (c.style = "label" /\ c.explode /\ v.shape # "scalar" /\ Policy # "label-pct-dot")
       => \A s \in Items(v) : ~Contains(s, ".")

QueryKV(c, k, s) == <<<<k>>, EQ(s)>>
QKey(v, k) == IF v.shape = "map" THEN EQ(k) ELSE k
QueryEncode(c, v) ==
  CASE c.style = "form" /\ c.explode ->
         CASE v.shape = "scalar" -> <<QueryKV(c, "n", v.s)>>
           [] v.shape = "sequence" -> Map(LAMBDA s: QueryKV(c, "n", s), v.l)
           [] OTHER -> Map(LAMBDA e: <<QKey(v, e[1]), EQ(e[2])>>, Entries(v))
    [] c.style = "deepObject" ->
         Map(LAMBDA e: <<<<"n", "%5B">> \o QKey(v, e[1]) \o <<"%5D">>, EQ(e[2])>>, Entries(v))
    [] OTHER ->
         LET d == CASE c.style = "spaceDelimited" -> <<"%20">> [] c.style = "pipeDelimited" -> <<"%7C">>
                    [] OTHER -> <<",">>
             body == CASE v.shape = "scalar" -> EQ(v.s)
                       [] v.shape = "sequence" -> Join(Map(LAMBDA s: EQ(s), v.l), d)
                       [] OTHER -> Join(Map(LAMBDA e: QKey(v, e[1]) \o d \o EQ(e[2]), Entries(v)), d)
         IN <<<<<<"n">>, body>>>>
\* form explode=false splits the raw value on literal "," before decoding
\* (OAS 3.2 Appendix C.2), so escaped commas are data. spaceDelimited and
\* pipeDelimited are decoded in a single pass (Appendix E.6), so their
\* delimiter character cannot appear in data.
QueryAdmit(c, v) ==
  /\ c.style \in {"spaceDelimited", "pipeDelimited"} /\ Policy # "no-delim-check"
       => \A s \in Items(v) : ~Contains(s, Delim(c))
  /\ (SingletonCollides(c) /\ v.shape = "sequence") => v.l # << <<>> >>

HeaderEncode(c, v) ==
  CASE v.shape = "scalar" -> v.s
    [] v.shape = "sequence" -> Join(v.l, <<",", " ">>)
    [] OTHER -> Join(Map(LAMBDA e: IF c.explode THEN e[1] \o <<"=">> \o e[2] ELSE e[1] \o <<",">> \o e[2],
                         Entries(v)), <<",", " ">>)
\* Exploded members are "k=v": an empty or leading-space value survives the
\* RFC 9110 list framing, a trailing space does not. Exploded map keys end at
\* the first "=" and must not begin with whitespace.
HeaderAdmit(c, v) ==
  IF v.shape = "scalar" THEN Policy = "header-scalar-ows" \/ Trim(v.s) = v.s
  ELSE IF v.shape \in {"object", "map"} /\ c.explode
  THEN /\ \A s \in Items(v) : ~Contains(s, ",") /\ TrimR(s) = s
       /\ Policy # "map-key-unchecked" => \A k \in MapKeys(v) : ~Contains(k, "=") /\ TrimL(k) = k
  ELSE \A s \in Items(v) : s # <<>> /\ ~Contains(s, ",") /\ Trim(s) = s

CookieVal(c, s) == IF c.style = "form" THEN EP(s) ELSE s
CookiePairs(c, v) ==
  CASE v.shape = "scalar" -> << <<<<"n">>, CookieVal(c, v.s)>> >>
    [] c.style = "cookie" /\ c.explode /\ v.shape = "sequence" ->
         Map(LAMBDA s: <<<<"n">>, s>>, v.l)
    [] c.style = "cookie" /\ c.explode -> Entries(v)
    [] v.shape = "sequence" -> << <<<<"n">>, Join(Map(LAMBDA s: CookieVal(c, s), v.l), <<",">>)>> >>
    [] OTHER -> << <<<<"n">>, Join(Map(LAMBDA e: (IF v.shape = "map" THEN CookieVal(c, e[1]) ELSE e[1])
                                                  \o <<",">> \o CookieVal(c, e[2]), Entries(v)), <<",">>)>> >>
\* Rejected candidate: net/http AddCookie wraps values containing SP or ",".
Quote(t) == IF Policy = "go-quote" /\ (Contains(t, " ") \/ Contains(t, ","))
            THEN <<"\"">> \o t \o <<"\"">> ELSE t
CookieLine(c, v) == Join(Map(LAMBDA p: p[1] \o <<"=">> \o Quote(p[2]), CookiePairs(c, v)), <<";", " ">>)
\* RFC 6265 cookie-octets exclude ";" and "," (OAS 3.2 parameter example);
\* go-quote is the legacy encoder that admits them and relies on quoting.
SpecForbidden(c, v) == c.style = "cookie" /\ \E s \in Items(v) : Contains(s, ",")
CookieAdmit(c, v) ==
  /\ c.style = "cookie" /\ Policy # "go-quote" =>
       \* net/http validCookieValueByte: 0x20..0x7E except DQUOTE ";" "\";
       \* "," excluded by the OAS example; non-ASCII ("e") is outside.
       /\ \A s \in Items(v) : ~Contains(s, ";") /\ ~Contains(s, ",") /\ (Policy = "cookie-nonascii" \/ ~Contains(s, "e"))
       /\ \A i \in 1..Len(CookiePairs(c, v)) : Trim(CookiePairs(c, v)[i][2]) = CookiePairs(c, v)[i][2]
       \* Exploded map keys become cookie names: nonempty RFC 9110 tokens.
       /\ (c.explode /\ v.shape = "map" /\ Policy # "map-key-unchecked") => \A k \in MapKeys(v) : k # <<>> /\ \A i \in 1..Len(k) : TChar(k[i])
  /\ (SingletonCollides(c) /\ v.shape = "sequence") => v.l # << <<>> >>

RuleARejects(c, v) == NoKeyEmpty(c) /\ EmptyValue(v) /\ required

\* ---- transport ----
\* A normalizing intermediary decodes %2E inside a segment (RFC 3986 6.2.2.2).
\* Assumption: it preserves a complete protected dot segment; the model makes
\* no claim about intermediaries that then remove dot segments.
NormalizeDots(t) == IF Protected(t) THEN t ELSE [i \in 1..Len(t) |-> IF Pct2E(t[i]) THEN "." ELSE t[i]]
\* A foreign client may spell the protected escape with lowercase hex.
Lower(t) == [i \in 1..Len(t) |-> IF t[i] = "%2E" THEN "%2e" ELSE t[i]]
ParsePairs(line, pr) ==
  LET parts == SplitOn(line, ";")
      kept == SelectSeq(Map(LAMBDA p: Trim(p), parts), LAMBDA p: p # <<>>)
      strip(t) == IF pr = "go" /\ Len(t) >= 2 /\ t[1] = "\"" /\ t[Len(t)] = "\""
                  THEN SubSeq(t, 2, Len(t) - 1) ELSE t
      \* net/http keeps a leading space after "="; RFC 6265 5.2-style parsers trim.
      trimmed(t) == IF pr = "strict" THEN Trim(t) ELSE t
      \* Both peers drop pairs whose name is not an RFC 9110 token (net/http
      \* isToken) or whose value has a byte outside the cookie value set.
      named == SelectSeq(kept, LAMBDA p: ~Contains(After(p, "="), "e") /\ Before(p, "=") # <<>> /\
                         \A i \in 1..Len(Before(p, "=")) : Before(p, "=")[i] \in {"n", "p", "q", "a", ".", "|", "P"})
  IN Map(LAMBDA p: <<Before(p, "="), trimmed(strip(After(p, "=")))>>, named)

\* ---- decoders ----
Keyed(pairs, k) == SelectSeq(pairs, LAMBDA p: p[1] = <<k>>)
DecodeList(c, text, collides) ==
  IF text = <<>> /\ collides THEN Li(<<>>) ELSE Li(SplitOn(text, Delim(c)))
DecodeFlat(tokens, d, U(_)) ==
  LET parts == IF tokens = <<>> THEN <<>> ELSE SplitOn(tokens, d) IN
  IF Len(parts) % 2 = 1 THEN Reject
  ELSE LET keys == [i \in 1..(Len(parts) \div 2) |-> parts[2 * i - 1]]
       IN IF \E i \in 1..Len(keys) : keys[i] \notin {<<"p">>, <<"q">>} THEN Reject
          ELSE IF \E i, j \in 1..Len(keys) : i # j /\ keys[i] = keys[j] THEN Reject
          ELSE Ob([k \in {"p", "q"} |-> IF \E i \in 1..Len(keys) : keys[i] = <<k>>
                    THEN U(parts[2 * (CHOOSE i \in 1..Len(keys) : keys[i] = <<k>>)]) ELSE Gone])
DecodePairs(pairs, U(_)) ==
  IF \E i \in 1..Len(pairs) : pairs[i][1] \notin {<<"p">>, <<"q">>} THEN Reject
  ELSE IF \E i, j \in 1..Len(pairs) : i # j /\ pairs[i][1] = pairs[j][1] THEN Reject
  ELSE Ob([k \in {"p", "q"} |-> IF \E i \in 1..Len(pairs) : pairs[i][1] = <<k>>
            THEN U(pairs[CHOOSE i \in 1..Len(pairs) : pairs[i][1] = <<k>>][2]) ELSE Gone])
DecodeMapFlat(tokens, d, U(_)) ==
  LET parts == IF tokens = <<>> THEN <<>> ELSE SplitOn(tokens, d) IN
  IF Len(parts) % 2 = 1 THEN Reject
  ELSE LET es == [i \in 1..(Len(parts) \div 2) |-> <<U(parts[2 * i - 1]), U(parts[2 * i])>>]
       IN IF \E i, j \in 1..Len(es) : i # j /\ es[i][1] = es[j][1] THEN Reject ELSE Mp(es)
DecodeMapPairs(pairs, U(_)) ==
  LET es == Map(LAMBDA p: <<U(p[1]), U(p[2])>>, pairs) IN
  IF \E i, j \in 1..Len(es) : i # j /\ es[i][1] = es[j][1] THEN Reject ELSE Mp(es)
FlatOf(shape, tokens, d, U(_)) == IF shape = "map" THEN DecodeMapFlat(tokens, d, U) ELSE DecodeFlat(tokens, d, U)
PairsOf(shape, pairs, U(_)) == IF shape = "map" THEN DecodeMapPairs(pairs, U) ELSE DecodePairs(pairs, U)
EmptyOf(shape) == IF shape = "map" THEN Mp(<<>>) ELSE Ob(NoObj)
InnerKey(t) == IF Len(t) >= 3 /\ t[1] = "n" /\ t[2] = "[" /\ t[Len(t)] = "]" THEN SubSeq(t, 3, Len(t) - 1) ELSE <<"!">>

\* unprotect-late is the rejected candidate that recognizes the protected spelling
\* only after percent-decoding, so the literal text "%2E" decodes twice.
Late(t) == IF Policy = "unprotect-late" /\ t = <<"P">> THEN <<".">> ELSE t
PathDecode(c, shape, raw, n) ==
  LET seg == IF n THEN NormalizeDots(raw) ELSE raw
      \* Only a complete protected spelling is reversed, before the label prefix
      \* is removed and before delimiters split; unprotect-after-prefix is the
      \* rejected candidate that strips the prefix first.
      u == IF Policy = "unprotect-after-prefix" THEN seg ELSE Unprotect(seg)
      un(t) == IF Policy = "decode-first" THEN t ELSE Late(UnescAll(t))
      pre == CASE c.style = "label" -> IF u # <<>> /\ Head(u) = "." THEN Tail(u) ELSE <<"!">>
               [] OTHER -> u
      tokens == IF Policy = "decode-first" THEN UnescAll(pre) ELSE pre
  \* A literal or empty dot segment never reaches the route handler.
  IN IF DotSegment(seg) THEN Reject
     ELSE IF c.style = "matrix"
     THEN LET parts == Tail(SplitOn(tokens, ";")) IN
          CASE seg = <<>> \/ Head(seg) # ";" -> Reject
            [] shape = "scalar" -> IF parts = << <<"n">> >> THEN Sc(<<>>)
                                   ELSE IF Len(parts) = 1 /\ Before(parts[1], "=") = <<"n">> /\ Pos(parts[1], "=") > 0
                                   THEN Sc(un(After(parts[1], "="))) ELSE Reject
            [] shape = "sequence" ->
                 IF parts = << <<"n">> >> THEN Li(<<>>)
                 ELSE IF \E i \in 1..Len(parts) : Before(parts[i], "=") # <<"n">> \/ Pos(parts[i], "=") = 0 THEN Reject
                 ELSE IF c.explode THEN Li(Map(LAMBDA p: un(After(p, "=")), parts))
                 ELSE IF Len(parts) # 1 THEN Reject
                 ELSE Li(Map(LAMBDA s: un(s), SplitOn(After(parts[1], "="), ",")))
            [] OTHER ->
                 IF c.explode THEN PairsOf(shape, Map(LAMBDA p: <<Before(p, "="), After(p, "=")>>, parts),
                                           LAMBDA t: un(t))
                 ELSE IF parts = << <<"n">> >> THEN EmptyOf(shape)
                 ELSE IF Len(parts) # 1 \/ Before(parts[1], "=") # <<"n">> THEN Reject
                 ELSE FlatOf(shape, After(parts[1], "="), ",", LAMBDA t: un(t))
     ELSE CASE shape = "scalar" -> Sc(un(tokens))
            [] shape = "sequence" -> Li(Map(LAMBDA s: un(s), SplitOn(tokens, PathDelim(c))))
            [] OTHER ->
                 IF c.explode
                 THEN PairsOf(shape, Map(LAMBDA p: <<Before(p, "="), After(p, "=")>>, SplitOn(tokens, PathDelim(c))),
                              LAMBDA t: un(t))
                 ELSE FlatOf(shape, tokens, ",", LAMBDA t: un(t))

QueryDecode(c, shape, pairs) ==
  LET dec == Map(LAMBDA p: <<UnescAll(p[1]), UnescAll(p[2])>>, pairs)
      raw == Map(LAMBDA p: <<UnescAll(p[1]), p[2]>>, pairs)
      rawSplit == c.style = "form" /\ ~c.explode /\ Policy # "single-pass-comma" IN
  CASE c.style = "form" /\ c.explode ->
         CASE shape = "scalar" -> IF Keyed(dec, "n") = <<>> THEN Absent ELSE Sc(Keyed(dec, "n")[1][2])
           [] shape = "sequence" -> IF Keyed(dec, "n") = <<>> THEN Absent ELSE Li(Map(LAMBDA p: p[2], Keyed(dec, "n")))
           [] OTHER -> IF dec = <<>> THEN Absent ELSE PairsOf(shape, dec, LAMBDA t: t)
    [] c.style = "deepObject" ->
         IF dec = <<>> THEN Absent
         ELSE PairsOf(shape, Map(LAMBDA p: <<InnerKey(p[1]), p[2]>>, dec), LAMBDA t: t)
    [] OTHER ->
         IF Keyed(dec, "n") = <<>> THEN Absent
         ELSE IF Len(Keyed(dec, "n")) > 1 THEN Reject
         ELSE IF rawSplit
         THEN LET rt == Keyed(raw, "n")[1][2] IN
              CASE shape = "scalar" -> Sc(UnescAll(rt))
                [] shape = "sequence" ->
                     IF rt = <<>> THEN Li(<<>>) ELSE Li(Map(LAMBDA t: UnescAll(t), SplitOn(rt, ",")))
                [] OTHER -> FlatOf(shape, rt, ",", LAMBDA t: UnescAll(t))
         ELSE LET text == Keyed(dec, "n")[1][2] IN
              CASE shape = "scalar" -> Sc(text)
                [] shape = "sequence" -> DecodeList(c, text, SingletonCollides(c))
                [] OTHER -> FlatOf(shape, text, Delim(c), LAMBDA t: t)

HeaderDecode(c, shape, field) ==
  LET items == SelectSeq(Map(LAMBDA s: Trim(s), SplitOn(Trim(field), ",")), LAMBDA s: s # <<>>) IN
  CASE shape = "scalar" -> Sc(field)
    [] shape = "sequence" -> Li(items)
    [] c.explode -> PairsOf(shape, Map(LAMBDA s: <<Before(s, "="), After(s, "=")>>, items), LAMBDA t: t)
    [] OTHER -> FlatOf(shape, Join(items, <<",">>), ",", LAMBDA t: t)

CookieDecode(c, shape, pairs) ==
  LET un(t) == IF c.style = "form" THEN UnescAll(t) ELSE t IN
  CASE c.style = "cookie" /\ c.explode /\ shape = "sequence" ->
         IF Keyed(pairs, "n") = <<>> THEN Absent ELSE Li(Map(LAMBDA p: p[2], Keyed(pairs, "n")))
    [] c.style = "cookie" /\ c.explode /\ shape = "map" ->
         IF pairs = <<>> THEN Absent ELSE DecodeMapPairs(pairs, LAMBDA t: t)
    [] c.style = "cookie" /\ c.explode /\ shape = "object" ->
         LET own == SelectSeq(pairs, LAMBDA p: p[1] \in {<<"p">>, <<"q">>}) IN
         IF own = <<>> THEN Absent ELSE DecodePairs(own, LAMBDA t: t)
    [] OTHER ->
         IF Keyed(pairs, "n") = <<>> THEN Absent
         ELSE LET text == Keyed(pairs, "n")[1][2] IN
              CASE shape = "scalar" -> Sc(un(text))
                [] shape = "sequence" ->
                     IF text = <<>> THEN Li(<<>>) ELSE Li(Map(LAMBDA s: un(s), SplitOn(text, ",")))
                [] OTHER -> FlatOf(shape, text, ",", LAMBDA t: un(t))

Encode(c, v) ==
  CASE c.loc = "path" -> [ok |-> PathAdmit(c, v, PathSpell(c, v)), w |-> Protect(PathSpell(c, v)), p |-> <<>>]
    [] c.loc = "query" -> [ok |-> QueryAdmit(c, v), w |-> <<>>, p |-> QueryEncode(c, v)]
    [] c.loc = "header" -> [ok |-> HeaderAdmit(c, v), w |-> HeaderEncode(c, v), p |-> <<>>]
    [] OTHER -> [ok |-> CookieAdmit(c, v), w |-> CookieLine(c, v), p |-> <<>>]
Transport(c, e, pr, hc) ==
  CASE c.loc = "cookie" -> [w |-> <<>>, p |-> ParsePairs(e.w, pr)]
    [] c.loc = "header" -> [w |-> Trim(e.w), p |-> e.p]
    [] c.loc = "path" /\ hc = "lower" -> [w |-> Lower(e.w), p |-> e.p]
    [] OTHER -> [w |-> e.w, p |-> e.p]
Decode(c, shape, t, n) ==
  CASE c.loc = "path" -> PathDecode(c, shape, t.w, n)
    [] c.loc = "query" -> QueryDecode(c, shape, t.p)
    [] c.loc = "header" -> HeaderDecode(c, shape, t.w)
    [] OTHER -> CookieDecode(c, shape, t.p)
Expected(c, v) == IF NoKeyEmpty(c) /\ EmptyValue(v) THEN Absent ELSE v

Init == /\ codec \in Codecs
        /\ value \in Values(codec)
        /\ required \in BOOLEAN /\ normalize \in BOOLEAN /\ peer \in {"go", "strict"}
        /\ hexcase \in IF codec.loc = "path" THEN {"upper", "lower"} ELSE {"upper"}
        /\ phase = "encode" /\ admitted = FALSE
        /\ wire = [w |-> <<>>, p |-> <<>>] /\ decoded = Absent

EncodeStep == /\ phase = "encode"
              /\ LET e == Encode(codec, value) IN
                 /\ admitted' = (e.ok /\ ~(RuleARejects(codec, value) /\ Policy # "no-rule-a"))
                 /\ wire' = [w |-> e.w, p |-> e.p]
              /\ phase' = "transport"
              /\ UNCHANGED <<codec, value, required, normalize, hexcase, peer, decoded>>
TransportStep == /\ phase = "transport"
                 /\ wire' = Transport(codec, wire, peer, hexcase)
                 /\ phase' = "decode"
                 /\ UNCHANGED <<codec, value, required, normalize, hexcase, peer, admitted, decoded>>
DecodeStep == /\ phase = "decode"
              /\ decoded' = Decode(codec, value.shape, wire, normalize)
              /\ phase' = "done"
              /\ UNCHANGED <<codec, value, required, normalize, hexcase, peer, admitted, wire>>
Next == EncodeStep \/ TransportStep \/ DecodeStep \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars

\* Every admitted value decodes, after the transport, to its stated state.
RoundTrip == phase = "done" /\ admitted => decoded = Expected(codec, value)
\* A required binding never reaches the server as absence.
RequiredPresent == phase = "done" /\ admitted /\ required => decoded # Absent
\* Every rejection is necessary: the unadmitted value's own spelling (with
\* complete dot segments protected) is unroutable or fails to decode in some
\* admitted environment (normalizing intermediary, lowercase hex, Go or strict
\* cookie peer), or the specification forbids it.
NaiveFails(c, v) ==
  LET e == Encode(c, v) IN
  \/ (c.loc = "path" /\ e.w = <<>>)
  \/ \E n \in BOOLEAN, pr \in {"go", "strict"}, hc \in {"upper", "lower"} :
       Decode(c, v.shape, Transport(c, e, pr, hc), n) # Expected(c, v)
DomainMaximal == phase = "done" /\ ~admitted /\ ~RuleARejects(codec, value) /\ ~SpecForbidden(codec, value)
                   => NaiveFails(codec, value)
=============================================================================
