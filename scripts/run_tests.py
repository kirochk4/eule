from lib import *

import os
import subprocess
import pathlib
import re

from itertools import chain
from typing import Iterable

TEMP_NAME = "__test_build.exe"
EXTS = ["eult"]
GO_WD = "./goeule"
GO_MAIN = "."
TEST_FOLDER = "../tests"

os.chdir(GO_WD)


def read_out(file: pathlib.Path) -> list[str]:
    out: list[str] = []
    with open(file, "r", encoding="utf-8") as f:
        for line in f.readlines():
            if match := re.search(r"\/\/ out:(.+)", line):
                out.append(match.group(1).strip())
    return out


def read_err(file: pathlib.Path) -> str:
    with open(file, "r", encoding="utf-8") as f:
        for line in f.readlines():
            if match := re.search(r"\/\/ err:(.+)", line):
                return match.group(1).strip()
    return ""


def run_script(file: pathlib.Path) -> tuple[str, str]:
    result = subprocess.run(
        [TEMP_NAME, file],
        capture_output=True,
        text=True,
        encoding="utf-8",
        timeout=2,
    )
    return result.stdout, result.stderr


def check_out(out: str, expect: list[str]) -> str | None:
    outs = out.splitlines()
    if outs != expect:
        if len(outs) != len(expect):
            return f"want {len(expect)} lines, got {len(outs)}"
        for i, line in enumerate(outs):
            if line != expect[i]:
                return f"expected '{expect[i]}', got '{line}'"


def check_err(err: str, expect: str) -> str | None:
    out = err.splitlines()[0]
    if expect != "..." and out != expect:
        return f"expected '{expect}', got '{out}'"


def read_files(folder: str, exts: list[str]) -> Iterable[pathlib.Path]:
    return chain(*[pathlib.Path(folder).rglob(f"*.{ext}") for ext in exts])


if __name__ == "__main__":
    os.system(f"go build -o {TEMP_NAME} {GO_MAIN}")

    try:
        print(cover("tests"))
        errors = 0
        for file in read_files(TEST_FOLDER, EXTS):
            out = read_out(file)
            err = read_err(file)

            stdout, stderr = run_script(file)

            if stderr:
                result = check_err(stderr, err)
            else:
                result = check_out(stdout, out)

            if not result:
                print(file, "-> ok")
            else:
                errors += 1
                print(file, "-> error:", result)

        print(cover("result"))
        print("ok" if errors == 0 else f"error [{errors}]")
    finally:
        os.unlink(TEMP_NAME)
