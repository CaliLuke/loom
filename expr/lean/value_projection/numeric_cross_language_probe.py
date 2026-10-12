"""Readiness-only exact JSON decoding with Python's decimal adapter.

Pass the task-owned output directory from numeric_factor_probe.go.txt. This
checks lexical coefficient/exponent preservation without the expensive optional
full Fraction comparison of the 699 KB hexadecimal-underflow specimen.
"""

import json
import sys
import time
from decimal import Decimal
from pathlib import Path

print(sys.version)
for name in (
    "integer", "hex_quarter", "decimal_tiny", "hex_tiny",
    "float64_point_one", "float32_point_one",
):
    started = time.perf_counter()
    token = (Path(sys.argv[1]) / f"{name}.json").read_text()
    value = json.loads(token, parse_float=Decimal, parse_int=Decimal)
    coefficient, _, exponent = token.partition("e")
    actual = value.as_tuple()
    assert "".join(str(digit) for digit in actual.digits) == coefficient
    assert actual.exponent == int(exponent or "0")
    print(name, "exact coefficient/exponent preserved", time.perf_counter() - started)
