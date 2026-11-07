/- A codec metadata slot is explicit at an occurrence or inherited. The same
   rule applies independently to contentType, style, explode and allowReserved. -/
namespace MediaEncodingContext

inductive Codec where
  | json | text | octets
  deriving DecidableEq, Repr

def inherit (outer inner : Option Codec) : Option Codec :=
  match outer with
  | some value => some value
  | none => inner

def resolve (value : Option Codec) : Codec := value.getD Codec.octets

def oldDescend (_outer inner : Option Codec) : Option Codec := inner

theorem old_loses_root_json :
    resolve (oldDescend (some Codec.json) none) ≠ Codec.json := by decide

theorem occurrence_wins (value : Codec) (inner : Option Codec) :
    inherit (some value) inner = some value := rfl

theorem nested_context (outer middle inner : Option Codec) :
    inherit (inherit outer middle) inner = inherit outer (inherit middle inner) := by
  cases outer <;> rfl

theorem explicit_json_survives (middle inner : Option Codec) :
    resolve (inherit (inherit (some Codec.json) middle) inner) = Codec.json := rfl

theorem absent_context_preserves_default (inner : Option Codec) :
    inherit none inner = inner := rfl

/- Source schema facts stay separate from authored Encoding Object slots.
   Option Bool models flag presence, including explicit false. The default is
   computed from the encoded value's source type, not an authored media label. -/
def oldFieldCodec (authored : Option Codec) (flag : Option Bool) (sourceDefault : Codec) : Codec :=
  match authored with
  | some value => value
  | none => if flag.isSome then Codec.text else sourceDefault

def fieldCodec (authored : Option Codec) (flag : Option Bool) (sourceDefault : Codec) : Codec :=
  if flag.isSome then Codec.text else authored.getD sourceDefault

theorem old_authored_label_hides_style :
    oldFieldCodec (some Codec.json) (some false) Codec.json ≠
      fieldCodec (some Codec.json) (some false) Codec.json := by decide

theorem flags_override_source_default (value : Bool) (sourceDefault : Codec) :
    fieldCodec none (some value) sourceDefault = Codec.text := rfl

theorem explicit_false_overrides_source_default :
    fieldCodec none (some false) Codec.json = Codec.text := rfl

theorem flags_override_authored_content (authored : Option Codec) (flag : Bool) (sourceDefault : Codec) :
    fieldCodec authored (some flag) sourceDefault = Codec.text := rfl

theorem absent_flags_keep_source_default (sourceDefault : Codec) :
    fieldCodec none none sourceDefault = sourceDefault := rfl

theorem absent_flags_keep_authored_content (value sourceDefault : Codec) :
    fieldCodec (some value) none sourceDefault = value := rfl

theorem branch_defaults_stay_independent (left right : Codec) :
    (fieldCodec none none left, fieldCodec none none right) = (left, right) := rfl

theorem authored_context_suppresses_both_defaults (flag : Bool) (left right : Codec) :
    (fieldCodec none (some flag) left, fieldCodec none (some flag) right) =
      (Codec.text, Codec.text) := rfl

end MediaEncodingContext
