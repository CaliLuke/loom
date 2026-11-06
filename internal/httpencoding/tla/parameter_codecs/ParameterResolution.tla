------------------------- MODULE ParameterResolution -------------------------
\* #968 binding codec resolution for path, query, header and cookie
\* parameters, checked against the OpenAPI 3.2 style tables and projected
\* through generated client/server, export (3.2 and 3.1) and re-import.
EXTENDS Naturals, FiniteSets
CONSTANT Policy
VARIABLES loc, shape, style, explode, reserved, media, plain, plainKeys, target, phase,
          codec, client, server, document, reimported
vars == <<loc, shape, style, explode, reserved, media, plain, plainKeys, target, phase,
          codec, client, server, document, reimported>>

\* "formfield" is a named application/x-www-form-urlencoded property whose
\* Encoding Object uses the query style table (OAS 3.2 §4.15.1.2).
\* reserved is the authored allowReserved ("absent", "true" or "false");
\* media is an explicit contentType (parameter content, or Encoding Object).
Locations == {"path", "query", "header", "cookie", "formfield"}
Shapes == {"scalar", "sequence", "object", "smap", "nmap", "nested", "union_ss", "union_so", "union_sm"}
Unions == {"union_ss", "union_so", "union_sm"}
Styles == {"matrix", "label", "simple", "form", "spaceDelimited", "pipeDelimited", "deepObject", "cookie"}
Reject == [style |-> "reject", explode |-> FALSE, shape |-> "none"]
JSON(sh) == [style |-> "json", explode |-> FALSE, shape |-> sh]
RejectDoc == [style |-> "reject", explode |-> "reject"]
NoDoc == [style |-> "none", explode |-> "none"]

\* ---- Independent oracle: OpenAPI 3.2 sections 4.12.3, 4.12.6 and D.1 ----
\* Style Values table: locations per style.
In == [matrix |-> {"path"}, label |-> {"path"}, simple |-> {"path", "header"},
       form |-> {"query", "cookie", "formfield"}, spaceDelimited |-> {"query", "formfield"},
       pipeDelimited |-> {"query", "formfield"}, deepObject |-> {"query", "formfield"}, cookie |-> {"cookie"}]
\* Style Examples table: the cells that are not n/a, as <<style, explode, column>>.
Cells == {<<s, e, c>> : s \in {"matrix", "label", "simple", "form", "cookie"},
                        e \in BOOLEAN, c \in {"primitive", "array", "object"}}
         \cup {<<s, FALSE, c>> : s \in {"spaceDelimited", "pipeDelimited"}, c \in {"array", "object"}}
         \cup {<<"deepObject", e, "object">> : e \in BOOLEAN}
Column(sh) == CASE sh = "scalar" -> "primitive" [] sh = "sequence" -> "array"
                [] sh \in {"object", "smap"} -> "object" [] OTHER -> "nested"
OASStyleDefault == [path |-> "simple", query |-> "form", header |-> "simple", cookie |-> "form", formfield |-> "form"]
OASExplodeDefault(s) == s \in {"form", "cookie"}
\* D.1: form with the default explode on non-primitive cookie values uses "&".
WrongDelimiter(l, s, e, sh) == l = "cookie" /\ s = "form" /\ e /\ Column(sh) # "primitive"
Defined(l, s, e, sh) == /\ l \in In[s]
                        /\ <<s, e, Column(sh)>> \in Cells
                        /\ ~WrongDelimiter(l, s, e, sh)
\* OAS 3.2 §4.12.2: a Parameter has either schema (with style, explode and
\* allowReserved) or content, never both.
\* OAS 3.2 §4.15.1.2: on an Encoding Object, an explicitly defined style,
\* explode or allowReserved, of any value, makes contentType (implicit or
\* explicit) ignored; without style, the style is "form".
Flags(s, e, r) == s # "absent" \/ e # "absent" \/ r # "absent"
URLLike(l) == l \in {"query", "formfield"}
\* Independent exposed top-level name computation (query keys / cookie names).
\* "*" is the residual namespace; "n[*]" is the deepObject prefix namespace.
Exposed(l, s, e, sh) ==
  CASE l \in {"path", "header"} -> {"n"}
    [] sh = "smap" /\ e /\ ((l = "cookie" /\ s = "cookie") \/ (URLLike(l) /\ s = "form")) -> {"*"}
    [] sh = "smap" /\ s = "deepObject" -> {"n[*]"}
    [] sh = "object" /\ s = "deepObject" -> {"n[p]", "n[q]"}
    [] sh = "object" /\ e /\ ((URLLike(l) /\ s = "form") \/ (l = "cookie" /\ s = "cookie")) -> {"p", "q"}
    [] OTHER -> {"n"}

\* ---- Design algorithm (Resolve pseudocode) ----
\* Shapes with an existing JSON carrier. Whole metadata-free query maps and
\* form-field objects/maps default to it; an explicit JSON content type also
\* selects it for unions (#1014's JSON matcher).
JSONDefaultShape(l, sh) == (l = "query" /\ sh \in {"smap", "nmap"}) \/ (l = "formfield" /\ sh \in {"object", "smap", "nmap"})
ContentCarrier(l, sh) == JSONDefaultShape(l, sh) \/ (URLLike(l) /\ sh \in {"union_ss", "union_so", "union_sm"})
\* Rejected candidate form-explicit-false-ignored treats an explicit false as absent.
FormFlags(s, e, r) == IF Policy = "form-explicit-false-ignored" THEN s # "absent" \/ e = "true" \/ r = "true"
                      ELSE Flags(s, e, r)
\* Codec family before style resolution: "json", "reject" or "style".
Kind(l, sh, s, e, r, m) ==
  CASE l = "query" /\ m = "json" ->
         \* encoding-on-parameters wrongly applies the Encoding Object rule to Parameter content.
         IF Flags(s, e, r) THEN (IF Policy = "encoding-on-parameters" THEN "style" ELSE "reject")
         ELSE IF ContentCarrier(l, sh) THEN "json" ELSE "reject"
    [] l = "query" /\ JSONDefaultShape(l, sh) /\ s = "absent" /\ e = "absent" ->
         \* allowReserved alone never switches the retained JSON codec; a content
         \* codec cannot carry it (reserved-switches-query is the rejected candidate).
         IF r = "absent" THEN "json" ELSE IF Policy = "reserved-switches-query" THEN "style" ELSE "reject"
    [] l = "formfield" /\ FormFlags(s, e, r) ->
         \* form-content-first keeps contentType; media-style-reject is the old blanket rejection.
         CASE Policy = "form-content-first" /\ (m = "json" \/ JSONDefaultShape(l, sh)) ->
                IF ContentCarrier(l, sh) THEN "json" ELSE "reject"
           [] Policy = "media-style-reject" /\ m = "json" -> "reject"
           [] OTHER -> "style"
    [] l = "formfield" ->
         IF JSONDefaultShape(l, sh) \/ (m = "json" /\ ContentCarrier(l, sh)) THEN "json"
         \* Typeless (union) properties default to application/octet-stream (#1014).
         ELSE IF m = "json" \/ sh \in {"union_ss", "union_so", "union_sm"} THEN "reject" ELSE "style"
    [] OTHER -> "style"
LoomStyleDefault == [path |-> "simple", query |-> "form", header |-> "simple", cookie |-> "cookie", formfield |-> "form"]
EffStyle(l, s) == IF s = "absent" THEN LoomStyleDefault[l] ELSE s
EffExplode(l, s, e) == IF e = "absent" THEN OASExplodeDefault(EffStyle(l, s)) ELSE e = "true"
Branch(sh, i) == CASE sh = "union_ss" -> IF i = 1 THEN "scalar" ELSE "sequence"
                   [] sh = "union_sm" -> IF i = 1 THEN "scalar" ELSE "smap"
                   [] OTHER -> IF i = 1 THEN "scalar" ELSE "object"
Erasing(l, s, e, sh) ==
  \/ s = "deepObject"
  \/ (sh \in {"object", "smap"} /\ e /\ ((URLLike(l) /\ s = "form") \/ (l = "cookie" /\ s = "cookie")))
ResolveOne(l, sh, s, e) ==
  IF s \notin {"matrix", "label", "simple", "form", "spaceDelimited", "pipeDelimited", "deepObject", "cookie"}
     \/ l \notin In[s] \/ sh \in {"nested", "nmap"} THEN Reject
  ELSE IF Policy = "reduced" /\
          ((s = "label" /\ e /\ sh # "scalar") \/ (l = "cookie" /\ sh \in {"object", "smap"})
           \/ (l = "cookie" /\ s = "form" /\ sh = "sequence"))
       THEN Reject
  ELSE IF Policy = "maps-json-only" /\ l = "query" /\ sh = "smap" THEN Reject
  ELSE CASE s = "deepObject" ->
              IF sh \in {"object", "smap"} \/ (Policy = "deep-arrays" /\ sh = "sequence")
              THEN [style |-> s, explode |-> FALSE, shape |-> sh] ELSE Reject
         [] s \in {"spaceDelimited", "pipeDelimited"} ->
              IF sh \in {"sequence", "object", "smap"} /\ ~e THEN [style |-> s, explode |-> FALSE, shape |-> sh] ELSE Reject
         [] s = "form" /\ l = "cookie" /\ sh # "scalar" /\ e -> Reject
         [] sh = "scalar" -> [style |-> s, explode |-> OASExplodeDefault(s), shape |-> sh]
         [] OTHER -> [style |-> s, explode |-> e, shape |-> sh]
RECURSIVE Resolve(_, _, _, _, _, _)
Resolve(l, sh, s, e, r, m) ==
  IF Kind(l, sh, s, e, r, m) = "json" THEN JSON(sh)
  ELSE IF Kind(l, sh, s, e, r, m) = "reject" THEN Reject
  ELSE LET st == EffStyle(l, s)
           ex == EffExplode(l, s, e)
       IN IF sh \in Unions
          \* The effective style is resolved once for the whole binding; the
          \* per-branch-default candidate lets each branch apply its own default.
          THEN LET b1 == ResolveOne(l, Branch(sh, 1), st, ex)
                   b2 == IF Policy = "per-branch-default" THEN Resolve(l, Branch(sh, 2), s, e, r, m)
                         ELSE ResolveOne(l, Branch(sh, 2), st, ex)
               \* union-name-only is the rejected candidate that refuses name-erasing
               \* branches instead of composing their KeySets.
               IN IF b1 = Reject \/ b2 = Reject
                     \/ (Policy = "union-name-only" /\
                         (Erasing(l, st, ex, Branch(sh, 1)) \/ Erasing(l, st, ex, Branch(sh, 2))))
                  THEN Reject
                  ELSE [style |-> IF b1.style = b2.style THEN st ELSE "mixed", explode |-> ex, shape |-> sh]
          ELSE ResolveOne(l, sh, st, ex)

\* Legacy runtime at f0c7a2b8c: styles are ignored; path/header are simple,
\* cookies raw scalars, query uses Select.
Legacy(l, sh) ==
  CASE l \in {"path", "header"} /\ sh \in {"scalar", "sequence"} -> [style |-> "simple", explode |-> FALSE, shape |-> sh]
    [] l = "cookie" /\ sh = "scalar" -> [style |-> "cookie", explode |-> TRUE, shape |-> sh]
    [] l = "query" /\ sh \in {"scalar", "sequence"} -> [style |-> "form", explode |-> TRUE, shape |-> sh]
    [] l = "query" /\ sh \in {"smap", "nmap"} -> JSON(sh)
    [] l = "formfield" /\ sh \in {"scalar", "sequence"} -> [style |-> "form", explode |-> TRUE, shape |-> sh]
    [] l = "formfield" /\ sh \in {"object", "smap", "nmap"} -> JSON(sh)
    [] OTHER -> Reject

\* ---- Projection algorithms ----
\* Export emits style/explode only where they differ from the OpenAPI default,
\* except that a styled form-field object, map or union always names its style:
\* without a style field its Encoding Object would select the contentType
\* (form-flag-dropped is the rejected candidate that omits it).
Field(present, value) == IF present THEN value ELSE "absent"
ExportFields(c, l) ==
  IF c.style = "json" THEN [style |-> "content", explode |-> "absent"]
  ELSE [style |-> Field(c.style # OASStyleDefault[l] \/
                        (l = "formfield" /\ c.shape \notin {"scalar", "sequence"} /\ Policy # "form-flag-dropped"),
                        c.style),
        explode |-> Field(c.style # "deepObject" /\ c.shape # "scalar" /\
                          c.explode # OASExplodeDefault(c.style),
                          IF c.explode THEN "true" ELSE "false")]
\* 3.1.1 can describe a cookie-style binding as form only when every key and
\* every value it can send is unreserved-only (form would not encode them).
\* form31-values-only is the rejected candidate that ignores keys.
KeysPlain(c) == c.shape \in {"scalar", "sequence"} \/ plainKeys \/ Policy = "form31-values-only"
CookieAsForm31(c) == plain /\ KeysPlain(c) /\ (c.shape = "scalar" \/ ~c.explode)
Export(c, l) ==
  IF c = Reject THEN RejectDoc
  ELSE IF Policy = "legacy"
       THEN [style |-> IF style = "absent" THEN "absent" ELSE style, explode |-> "absent"]
  ELSE IF target = "3.1" /\ c.style = "cookie"
       THEN IF Policy = "form31" \/ CookieAsForm31(c)
            THEN ExportFields([c EXCEPT !.style = "form"], l)
            ELSE RejectDoc
  ELSE ExportFields(c, l)
\* Import reads the document with OpenAPI defaults and renders metadata only
\* when the codec differs from Loom's default for that location and shape
\* (cookies default to "cookie"; named query maps default to JSON content).
LoomDefaultFor(l, sh) == IF (l = "query" /\ sh \in {"smap", "nmap"}) \/ (l = "formfield" /\ sh \notin {"scalar", "sequence"})
                         THEN "json" ELSE LoomStyleDefault[l]
\* A form field without style fields is read by its Encoding Object contentType
\* default (§4.15.1.1), not by the parameter style default.
ImportResolve(d, l, sh) ==
  IF d.style = "content" THEN Resolve(l, sh, "absent", "absent", "absent", "json")
  ELSE IF l = "formfield" /\ d.style = "absent" /\ d.explode = "absent"
       THEN Resolve(l, sh, "absent", "absent", "absent", "absent")
  ELSE LET s == IF d.style = "absent" THEN OASStyleDefault[l] ELSE d.style
           e == IF d.explode = "absent" THEN "absent" ELSE d.explode
           rs == IF s = LoomDefaultFor(l, sh) /\ e = "absent" THEN "absent" ELSE s
       IN Resolve(l, sh, rs, IF e = "absent" /\ rs # "absent" THEN
                               IF OASExplodeDefault(s) THEN "true" ELSE "false" ELSE e, "absent", "absent")
WireEq(a, b) == \/ a = b
                \/ /\ plain /\ (a.shape \in {"scalar", "sequence"} \/ plainKeys) /\ {a.style, b.style} = {"form", "cookie"}
                   /\ a.shape = b.shape /\ (a.shape = "scalar" \/ (~a.explode /\ ~b.explode))

Init == /\ loc \in Locations /\ shape \in Shapes
        /\ style \in Styles \cup {"absent"} /\ explode \in {"absent", "true", "false"}
        /\ reserved \in IF URLLike(loc) THEN {"absent", "true", "false"} ELSE {"absent"}
        /\ media \in IF URLLike(loc) THEN {"absent", "json"} ELSE {"absent"}
        /\ plain \in BOOLEAN /\ plainKeys \in BOOLEAN /\ target \in {"3.2", "3.1"}
        /\ phase = "author"
        /\ codec = Reject /\ client = Reject /\ server = Reject
        /\ document = NoDoc /\ reimported = Reject

Evaluate == /\ phase = "author"
            /\ codec' = IF Policy = "legacy" THEN Legacy(loc, shape) ELSE Resolve(loc, shape, style, explode, reserved, media)
            /\ phase' = "generate"
            /\ UNCHANGED <<loc, shape, style, explode, reserved, media, plain, plainKeys, target, client, server, document, reimported>>
Generate == /\ phase = "generate"
            /\ client' = codec /\ server' = codec
            /\ phase' = "export"
            /\ UNCHANGED <<loc, shape, style, explode, reserved, media, plain, plainKeys, target, codec, document, reimported>>
ExportStep == /\ phase = "export"
              /\ document' = Export(codec, loc)
              /\ phase' = IF document' = RejectDoc THEN "done" ELSE "import"
              /\ UNCHANGED <<loc, shape, style, explode, reserved, media, plain, plainKeys, target, codec, client, server, reimported>>
ImportStep == /\ phase = "import"
              /\ reimported' = ImportResolve(document, loc, shape)
              /\ phase' = "done"
              /\ UNCHANGED <<loc, shape, style, explode, reserved, media, plain, plainKeys, target, codec, client, server, document>>
Next == Evaluate \/ Generate \/ ExportStep \/ ImportStep \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars

AuthoredStyle == EffStyle(loc, style)
AuthoredExplode == EffExplode(loc, style, explode)
UnionDefined == shape \in Unions =>
                  /\ Defined(loc, AuthoredStyle, AuthoredExplode, Branch(shape, 1))
                  /\ Defined(loc, AuthoredStyle, AuthoredExplode, Branch(shape, 2))
ShapeDefined == IF shape \in Unions THEN UnionDefined
                ELSE Defined(loc, AuthoredStyle, AuthoredExplode, shape)

\* Nothing outside the specification's defined cells is admitted.
NoUndefinedAdmitted == phase # "author" /\ codec # Reject /\ codec.style # "json" => ShapeDefined
\* The binding is style-serialized: no Parameter content, or an Encoding Object
\* whose explicit style fields override contentType. Excluded: a metadata-free
\* query map whose only field is allowReserved (the retained JSON codec cannot
\* carry it), and a form-field object, map or union without style fields.
StyleSerialized ==
  CASE loc = "query" -> media = "absent" /\ ~(JSONDefaultShape(loc, shape) /\ style = "absent" /\ explode = "absent")
    [] loc = "formfield" -> Flags(style, explode, reserved) \/ (media = "absent" /\ shape \in {"scalar", "sequence"})
    [] OTHER -> TRUE
\* Every defined single-level cell is admitted (no silent scope reduction),
\* including scalar-valued maps and union branches under every defined style.
\* A metadata-free named query map is admitted as JSON.
FullScope == phase # "author" /\ StyleSerialized /\ ShapeDefined => codec # Reject
\* §4.15.1.2: explicit style fields, by presence, select the style over contentType.
EncodingStyleFirst == phase # "author" /\ loc = "formfield" /\ Flags(style, explode, reserved) /\ codec # Reject
                        => codec.style # "json"
\* §4.12.2: Parameter content excludes the schema-only style fields.
ParameterContentExclusive == phase # "author" /\ loc = "query" /\ media = "json" /\ Flags(style, explode, reserved)
                               => codec = Reject
\* allowReserved never changes a query binding's codec family.
QueryReservedNeutral == phase # "author" /\ loc = "query" /\ codec # Reject =>
                          (codec.style = "json") = (Resolve(loc, shape, style, explode, "absent", media).style = "json")
\* A union's KeySet is the union of its branch KeySets, and every name-erasing
\* branch contributes a nonempty namespace that endpoint collision checks see.
UnionKeySet(l, s, e, sh) == Exposed(l, s, e, Branch(sh, 1)) \cup Exposed(l, s, e, Branch(sh, 2))
\* One effective style and explode serve every branch of a union binding.
UnionOneStyle == phase # "author" /\ codec # Reject /\ shape \in Unions => codec.style # "mixed"
UnionKeysCover == phase # "author" /\ codec # Reject /\ shape \in Unions =>
                    \A i \in {1, 2} : Exposed(loc, codec.style, codec.explode, Branch(shape, i)) # {}
                       /\ Exposed(loc, codec.style, codec.explode, Branch(shape, i))
                            \subseteq UnionKeySet(loc, codec.style, codec.explode, shape)
\* An explicitly authored Loom default style is the same codec as absent
\* metadata. Named query maps default to a content codec, not a style.
DefaultEquivalence == ~(loc = "query" /\ shape \in {"smap", "nmap"})
                      /\ ~(loc = "formfield" /\ shape \notin {"scalar", "sequence"}) =>
  Resolve(loc, shape, LoomStyleDefault[loc],
          IF OASExplodeDefault(LoomStyleDefault[loc]) THEN "true" ELSE "false", "absent", "absent")
    = Resolve(loc, shape, "absent", "absent", "absent", "absent")
ClientServerAgree == phase \in {"export", "import", "done"} => client = server
\* The exported description, read back with OpenAPI defaults, is the same
\* wire as the generated client (or provably wire-identical on the domain).
ExportPreservesWire == phase = "done" /\ document # RejectDoc /\ client # Reject =>
                         WireEq(reimported, client)
=============================================================================
