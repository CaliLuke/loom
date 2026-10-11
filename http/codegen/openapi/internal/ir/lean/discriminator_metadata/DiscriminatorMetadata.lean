import Std
namespace Loom.OpenAPI.DiscriminatorMetadata

-- The legacy owner independently removed requiredness without a fallback check.
def legacyAdmits (_optional _fallback : Bool) : Bool := true
-- The new boundary admits optionality only with a resolved explicit fallback.
def admits (optional validFallback : Bool) : Bool := !optional || validFallback
-- Version 3.1 cannot represent fallback selection and requires the property.
def required31 (_optional : Bool) : Bool := true

theorem legacy_missing_fallback : legacyAdmits true false = true := rfl
theorem rejects_missing_fallback : admits true false = false := rfl
theorem admits_valid_fallback : admits true true = true := rfl
theorem preserves_required_control : admits false false = true := rfl
theorem projection31_requires_property : ∀ optional, required31 optional = true := by
  intro optional
  rfl

#print axioms legacy_missing_fallback
#print axioms rejects_missing_fallback
#print axioms admits_valid_fallback
#print axioms preserves_required_control
#print axioms projection31_requires_property
end Loom.OpenAPI.DiscriminatorMetadata
