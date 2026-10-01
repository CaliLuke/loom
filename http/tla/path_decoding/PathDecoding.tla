------------------------- MODULE PathDecoding -------------------------
EXTENDS Naturals, Sequences, FiniteSets
CONSTANTS LegacyArrays, LegacyLiterals
VARIABLES values, literal, rawRequired

Elements == {<<"a">>, <<"a", ",", "b">>, <<"/">>,
             <<"%", "2", "C">>, <<"+">>, <<>>}
Literals == {<<"plain">>, <<"my", " ", "files">>, <<"unicode">>}
EncodeToken(t) == CASE t = "," -> "%2C"
                   [] t = "/" -> "%2F"
                   [] t = "%" -> "%25"
                   [] t = " " -> "%20"
                   [] t = "unicode" -> "%UTF8"
                   [] OTHER -> t
DecodeToken(t) == CASE t = "%2C" -> ","
                   [] t = "%2F" -> "/"
                   [] t = "%25" -> "%"
                   [] t = "%20" -> " "
                   [] t = "%UTF8" -> "unicode"
                   [] OTHER -> t
Encode(s) == [i \in 1..Len(s) |-> EncodeToken(s[i])]
Decode(s) == [i \in 1..Len(s) |-> DecodeToken(s[i])]

RECURSIVE Join(_), Split(_)
Join(items) == IF Len(items) = 1 THEN Head(items)
               ELSE Head(items) \o <<",">> \o Join(Tail(items))
Split(tokens) ==
  IF Len(tokens) = 0 THEN << <<>> >>
  ELSE LET rest == Split(Tail(tokens))
       IN IF Head(tokens) = "," THEN << <<>> >> \o rest
          ELSE << <<Head(tokens)>> \o Head(rest) >> \o Tail(rest)

Wire == Join([i \in 1..Len(values) |-> Encode(values[i])])
Chunks == Split(IF LegacyArrays THEN Decode(Wire) ELSE Wire)
Decoded == IF LegacyArrays THEN Chunks
           ELSE [i \in 1..Len(Chunks) |-> Decode(Chunks[i])]
MatchedLiteral == IF LegacyLiterals /\ ~rawRequired THEN literal ELSE Encode(literal)
RegisteredLiteral == IF LegacyLiterals THEN literal ELSE Encode(literal)

Init == /\ values \in UNION {[1..n -> Elements]: n \in 1..2}
        /\ literal \in Literals
        /\ rawRequired \in BOOLEAN
Next == UNCHANGED <<values, literal, rawRequired>>
Spec == Init /\ [][Next]_<<values, literal, rawRequired>>
ArrayRoundTrip == Decoded = values
LiteralMatches == MatchedLiteral = RegisteredLiteral
========================================================================
