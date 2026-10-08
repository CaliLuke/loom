"""Regenerate and exercise the pinned OAuth fixture, without editing generated code."""
from pathlib import Path
import os
import shutil
import subprocess
import tempfile

HERE = Path(__file__).resolve().parent
MODULE = "example.com/formunionit"


def run(*args, cwd=HERE):
    subprocess.run(args, cwd=cwd, check=True, env={**os.environ, "GOFLAGS": "-mod=mod"})


def main():
    source = Path(subprocess.check_output(
        ["go", "list", "-m", "-f", "{{.Dir}}", "github.com/CaliLuke/loom"],
        cwd=HERE, text=True).strip())
    fixture = (source / "http/codegen/form_union_oauth_integration_test.go").read_text()
    design = fixture.split("func oauthFormRequestUnionDSL() {", 1)[1].split(
        "\nconst oauthIntegrationHarness =", 1)[0]
    harness = fixture.split("const oauthIntegrationHarness = `", 1)[1].rsplit("`", 1)[0]
    with tempfile.TemporaryDirectory(prefix="loom-form-oauth-") as tmp:
        root = Path(tmp)
        (root / "design").mkdir()
        (root / "design/design.go").write_text(
            'package design\nimport . "github.com/CaliLuke/loom/dsl"\nfunc init() {' + design)
        mod = (HERE / "go.mod").read_text().replace(
            "module github.com/CaliLuke/loom/http/testdata/form_comparison",
            "module " + MODULE)
        (root / "go.mod").write_text(mod)
        shutil.copy(HERE / "go.sum", root / "go.sum")
        (root / "baseline_test.go").write_text(harness)
        shutil.copy(HERE / "oauth_adapter_test.go.txt", root / "adapter_test.go")
        run("go", "mod", "edit", "-require=golang.org/x/oauth2@v0.37.0", cwd=root)
        run("go", "run", "-mod=mod", "github.com/CaliLuke/loom/cmd/loom", "gen",
            MODULE + "/design", "-o", ".", cwd=root)
        run("go", "mod", "tidy", cwd=root)
        run("go", "test", "-race", "-count=1", "-v", "./...", cwd=root)
        run("go", "vet", "./...", cwd=root)


if __name__ == "__main__":
    main()
