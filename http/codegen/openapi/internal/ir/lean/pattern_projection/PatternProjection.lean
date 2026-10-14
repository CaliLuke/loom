import Std

namespace Loom.OpenAPI.PatternProjection

-- Finite witness model: 0=foo, 1=Afooz, 2=foo+LF, 3=space, 4=tab,
-- 5=vertical tab, 6=no-break space. Pattern 0 is absolute foo; 1 is whitespace.
def goAccepts (pattern witness : Nat) : Bool :=
  (pattern == 0 && witness == 0) || (pattern == 1 && (witness == 3 || witness == 4))

def unchangedECMA (pattern witness : Nat) : Bool :=
  (pattern == 0 && witness == 1) ||
  (pattern == 1 && (witness == 3 || witness == 4 || witness == 5 || witness == 6))

-- ^foo$ and [ \t\n\f\r] in an ECMA engine without flags.
def projectedECMA (pattern witness : Nat) : Bool :=
  (pattern == 0 && witness == 0) || (pattern == 1 && (witness == 3 || witness == 4))

theorem legacy_anchor_counterexample :
    (goAccepts 0 0 == unchangedECMA 0 0) = false := rfl

theorem legacy_whitespace_counterexample :
    (goAccepts 1 6 == unchangedECMA 1 6) = false := rfl

theorem bounded_projection_preserves :
    ∀ p w, goAccepts p w = projectedECMA p w := by
  intro p w
  rfl

#print axioms legacy_anchor_counterexample
#print axioms legacy_whitespace_counterexample
#print axioms bounded_projection_preserves

-- Importing authored ECMA whitespace requires the larger whitespace language.
-- The witness domain and engine/parser limitations above remain unchanged.
def authoredWhitespace (witness : Nat) : Bool :=
  witness == 3 || witness == 4 || witness == 5 || witness == 6

def translatedWhitespace (witness : Nat) : Bool :=
  witness == 3 || witness == 4 || witness == 5 || witness == 6

theorem import_identity_whitespace_counterexample :
    (authoredWhitespace 6 == goAccepts 1 6) = false := rfl

theorem bounded_import_translation_preserves :
    ∀ w, authoredWhitespace w = translatedWhitespace w := by
  intro w
  rfl

#print axioms import_identity_whitespace_counterexample
#print axioms bounded_import_translation_preserves

end Loom.OpenAPI.PatternProjection
