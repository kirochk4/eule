def short(string: str, length: int) -> str:
    if len(string) > length:
        return string[:length]
    return string


def cover(string: str, width: int = 24, char: str = "=") -> str:
    to_cover = width - len(string) - 2
    if to_cover < 0:
        return string
    left = to_cover // 2
    right =  left + to_cover % 2
    return f"{left * char} {string} {right * char}"
