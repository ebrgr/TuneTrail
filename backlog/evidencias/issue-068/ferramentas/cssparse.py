"""Tiny CSS chunker: returns (raw, prelude, body, line) per top-level rule; trailing text as (raw, None, None, line)."""
import re


def chunks(text, base_line=1):
    i, n = 0, len(text)
    out = []
    while i < n:
        start = i
        while True:
            m = re.match(r'\s*(/\*.*?\*/)?', text[i:], re.S)
            if not m.group(0):
                break
            i += len(m.group(0))
            if not m.group(1):
                break
        if i >= n:
            out.append((text[start:], None, None, None))
            break
        brace = text.index('{', i)
        depth, j = 0, brace
        while True:
            if text.startswith('/*', j):
                j = text.index('*/', j) + 2
                continue
            c = text[j]
            if c == '{':
                depth += 1
            elif c == '}':
                depth -= 1
                if depth == 0:
                    break
            elif c in '"\'':
                j = text.index(c, j + 1)
            j += 1
        out.append((text[start:j + 1], text[i:brace].strip(), text[brace + 1:j], base_line + text.count('\n', 0, i)))
        i = j + 1
    return out
