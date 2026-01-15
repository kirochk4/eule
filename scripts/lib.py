import subprocess
import pathlib

from itertools import chain
from typing import Iterable


def short(string: str, length: int) -> str:
    if len(string) > length:
        return string[:length]
    return string


def cover(string: str, width: int = 24, char: str = "=") -> str:
    to_cover = width - len(string) - 2
    if to_cover < 0:
        return string
    left = to_cover // 2
    right = left + to_cover % 2
    return f"{left * char} {string} {right * char}"


def run_script(
    executable: str, file: pathlib.Path, timeout: int = None
) -> tuple[str, str]:
    result = subprocess.run(
        [executable, file],
        capture_output=True,
        text=True,
        encoding="utf-8",
        timeout=timeout,
    )
    return result.stdout, result.stderr


def read_files(folder: str, exts: list[str]) -> Iterable[pathlib.Path]:
    return chain(*[pathlib.Path(folder).rglob(f"*.{ext}") for ext in exts])
