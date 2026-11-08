--------------------------- MODULE FragmentPointer ---------------------------
\* Fragment decoding and canonical emission for resolver target keys (#957).
\* RFC 6901 §6: percent-decode the URI fragment, then evaluate the JSON Pointer
\* (split on "/", then unescape "~1" before "~0" per token).
EXTENDS Naturals, Sequences, TLC

CONSTANT Mode
ASSUME Mode \in {"checked", "splitFirst", "escapeOrder"}

\* Fragment units: literal characters and percent-encoded triplets.
Units == {"a", "/", "~", "0", "1", "%2F", "%7E", "%7e", "%61"}
Chars == {"a", "/", "~"}

DecodeUnit(u) == CASE u = "%2F" -> "/" [] u \in {"%7E", "%7e"} -> "~"
                   [] u = "%61" -> "a" [] OTHER -> u
PercentDecode(f) == [i \in 1..Len(f) |-> DecodeUnit(f[i])]
EncodeUnit(c) == CASE c = "/" -> "%2F" [] c = "~" -> "%7E" [] c = "a" -> "%61" [] OTHER -> c
PercentEncodeAll(f) == [i \in 1..Len(PercentDecode(f)) |-> EncodeUnit(PercentDecode(f)[i])]

Bad == [ok |-> FALSE, v |-> <<>>]
Good(v) == [ok |-> TRUE, v |-> v]

\* Split a sequence on separator units; the input must start with a separator.
RECURSIVE SplitAcc(_, _, _, _)
SplitAcc(cs, sep, cur, acc) ==
    IF cs = <<>> THEN Append(acc, cur)
    ELSE IF Head(cs) = sep THEN SplitAcc(Tail(cs), sep, <<>>, Append(acc, cur))
    ELSE SplitAcc(Tail(cs), sep, Append(cur, Head(cs)), acc)
Split(cs, sep) ==
    IF cs = <<>> THEN Good(<<>>)
    ELSE IF Head(cs) # sep THEN Bad
    ELSE Good(SplitAcc(Tail(cs), sep, <<>>, <<>>))

RECURSIVE Unescape(_)
Unescape(t) ==
    IF t = <<>> THEN Good(<<>>)
    ELSE IF Head(t) # "~" THEN
        LET r == Unescape(Tail(t)) IN IF r.ok THEN Good(<<Head(t)>> \o r.v) ELSE Bad
    ELSE IF Len(t) >= 2 /\ t[2] \in {"0", "1"} THEN
        LET r == Unescape(Tail(Tail(t)))
        IN IF r.ok THEN Good(<<IF t[2] = "0" THEN "~" ELSE "/">> \o r.v) ELSE Bad
    ELSE Bad

RECURSIVE UnescapeAll(_)
UnescapeAll(ts) ==
    IF ts = <<>> THEN Good(<<>>)
    ELSE LET h == Unescape(Head(ts)) r == UnescapeAll(Tail(ts))
         IN IF h.ok /\ r.ok THEN Good(<<h.v>> \o r.v) ELSE Bad

\* splitFirst is the rejected order: split literal "/" units, then percent-decode.
Decode(f) ==
    IF Mode = "splitFirst"
    THEN LET s == Split(f, "/")
         IN IF s.ok THEN UnescapeAll([i \in 1..Len(s.v) |-> PercentDecode(s.v[i])]) ELSE Bad
    ELSE LET s == Split(PercentDecode(f), "/")
         IN IF s.ok THEN UnescapeAll(s.v) ELSE Bad

\* Rewrite each character in one pass; pass "slash" or "tilde" rewrites one kind.
EscChar(c, pass) ==
    CASE c = "~" /\ pass \in {"once", "tilde"} -> <<"~", "0">>
      [] c = "/" /\ pass \in {"once", "slash"} -> <<"~", "1">>
      [] OTHER -> <<c>>
RECURSIVE EscPass(_, _)
EscPass(t, pass) == IF t = <<>> THEN <<>> ELSE EscChar(Head(t), pass) \o EscPass(Tail(t), pass)
\* escapeOrder rewrites "/" before "~", corrupting the "~" it introduced.
Escape(t) == IF Mode = "escapeOrder" THEN EscPass(EscPass(t, "slash"), "tilde")
             ELSE EscPass(t, "once")
RECURSIVE Canonical(_)
Canonical(ts) == IF ts = <<>> THEN <<>> ELSE <<"/">> \o Escape(Head(ts)) \o Canonical(Tail(ts))

SeqsUpTo(S, n) == UNION {[1..k -> S] : k \in 0..n}

VARIABLES fragment, tokens
vars == <<fragment, tokens>>
Init == /\ fragment \in SeqsUpTo(Units, 4)
        /\ tokens \in SeqsUpTo(SeqsUpTo(Chars, 2), 2)
Next == UNCHANGED vars
Spec == Init /\ [][Next]_vars

\* Canonical emission round-trips every token sequence.
RoundTrip == Decode(Canonical(tokens)) = Good(tokens)
\* Re-emitting a decoded key preserves it.
KeyStable == Decode(fragment).ok => Decode(Canonical(Decode(fragment).v)) = Decode(fragment)
\* Percent-encoding any fragment character never changes the target key.
PercentInvariant == Decode(fragment).ok => Decode(PercentEncodeAll(fragment)) = Decode(fragment)
=============================================================================
