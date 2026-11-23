"""Extrai padrões HTTP canônicos do checkout. Saída TSV: método, caminho, registro, handler."""
import argparse
from pathlib import Path
import re

ROOT = Path(__file__).resolve().parents[3]
rows = {}


def source(file):
    return (ROOT / file).read_text()


def loc(file, offset):
    return f"{file}:{source(file).count(chr(10), 0, offset) + 1}"


def definition(name, scope):
    pattern = rf"^func (?:\([^\n]+?\) )?{name}\("
    for file in sorted(ROOT.glob(scope)):
        relative = file.relative_to(ROOT).as_posix()
        match = re.search(pattern, source(relative), re.M)
        if match:
            return loc(relative, match.start())
    raise ValueError((name, scope))


def add(method, path, registration, handler):
    key = method.upper(), path
    assert key not in rows, key
    rows[key] = registration, handler


def extract():
    file = "engine/routes.go"
    go = source(file)
    registrations = re.compile(r'\b(Get|Post|Patch|Put|Delete|Handle)\("([^"]+)",\s*([^\n]+)')
    scopes = []
    for match in re.finditer(r'^(\t+)r\.Route\("([^"]+)", func\(r chi.Router\) \{', go, re.M):
        end = re.search(rf'^\t{{{len(match[1])}}}\}}\)', go[match.end():], re.M)
        scopes.append((match.start(), match.end() + end.end(), match[2]))
    scopes.sort()
    for match in registrations.finditer(go):
        method, path, expression = match.groups()
        if path == "/*" or path in ("/admin/*", "/{slug}/manage/{kind}/{id}/{action}"):
            continue
        position = match.start()
        prefix = "".join(p for start, end, p in scopes if start <= position < end)
        prefixes = [prefix]
        if position > go.index("func bindProfileReads"):
            prefixes = ["/apis/web/v1", "/apis/web/v1/profiles/{slug}"]
        reg = loc(file, position)
        name = re.match(r'handlers\.(\w+)\(', expression)
        handler = definition(name[1], "engine/handlers/*.go") if name else reg
        if expression.startswith("profileImportHandler"):
            handler = definition("profileImportHandler", "engine/*.go")
        if expression.startswith("catalog.SetLanguage"):
            handler = definition("SetLanguage", "internal/i18n/*.go")
        if path in ("/web-assets/*", "/favicon.svg"):
            handler = definition("Handler", "internal/webassets/http.go")
        if path.startswith("/image/"):
            handler = definition("ImageHandler", "engine/handlers/*.go")
        for prefix in prefixes:
            add("GET/HEAD" if method == "Handle" else method,
                prefix + ("" if prefix.endswith("{slug}") and path == "/" else path), reg, handler)
    match = re.search(r'publicServer\(r, "([^"]+)"', go)
    add("GET", match[1] + "/*", loc(file, match.start()), definition("Handler", "internal/webassets/http.go"))

    admin = "internal/adminui/routing.go"
    web = "internal/webui/routing.go"
    admin_handler = definition("dispatch", "internal/adminui/admin.go")
    web_handler = definition("ServeHTTP", "internal/webui/handlers.go")

    def literals(file, start, end, prefix, method, handler, needle=None):
        body = source(file)
        first = body.index(start)
        last = body.index(end, first + len(start)) if end else len(body)
        for match in re.finditer(r'"([^"]+)"', body[first:last]):
            if needle and not match[1].startswith(needle):
                continue
            add(method, prefix + match[1], loc(file, first + match.start()), handler)

    pages = {
        "/admin": admin_handler,
        "/admin/health": definition("healthPage", "internal/adminui/ingest_health.go"),
        "/admin/settings": definition("settingsPage", "internal/adminui/settings.go"),
        "/admin/redeem": definition("redeemAccount", "internal/adminui/account_tokens.go"),
        "/admin/sign-out": definition("signOutPage", "internal/adminui/sign_out.go"),
    }
    for path, handler in pages.items():
        add("GET", path, loc(admin, source(admin).index(f'"{path}"')), handler)
    nav = "internal/webassets/access_nav.go"
    match = re.search(r'AccountDestination\s*=\s*"([^"]+)"', source(nav))
    add("GET", match[1], definition("IsAccountPath", nav), definition("accountPage", "internal/adminui/account.go"))
    for field in ("Profile", "Keys", "History", "Relay"):
        value = re.search(rf'Tab{field}\s*=\s*"([^"]+)"', source(nav))
        assert f'AccountDestination + "/" + Tab{field}' in source(nav)
        add("GET", match[1] + "/" + value[1], loc(nav, value.start()), definition("accountTab", "internal/adminui/account_tabs.go"))
    for suffix in ("export", "delete-review"):
        add("GET", "/admin/profile/{id}/" + suffix, loc(admin, source(admin).index('p[3] == "' + suffix)), admin_handler)
    actions = definition("action", "internal/adminui/actions.go")
    literals(admin, 'if path == "/admin/redeem" || path == "/admin/login"', "return true", "", "POST", actions, "/")
    rows["POST", "/admin/login"] = rows["POST", "/admin/login"][0], definition("postLogin", "internal/adminui/*.go")
    rows["POST", "/admin/redeem"] = rows["POST", "/admin/redeem"][0], pages["/admin/redeem"]
    literals(admin, 'case "save"', "return true", "/admin/profile/{id}/", "POST", actions)
    literals(admin, 'case "keys/create"', "return true", "/admin/profile/{id}/", "POST", actions)
    literals(admin, 'p[5] == "label"', "\n\t}", "/admin/profile/{id}/keys/{key_id}/", "POST", actions)
    literals(admin, 'case "/admin/health/discard"', "return true", "", "POST", definition("healthPage", "internal/adminui/ingest_health.go"))

    media = definition("media", "internal/adminui/media.go")
    ret = "internal/adminweb/return.go"
    for kind in ("artist", "album", "track"):
        prefix = "/{slug}/manage/" + kind + "/{id}"
        origin = loc(admin, source(admin).index('if (len(p) == 4'))
        add("GET", prefix, origin, media)
        add("GET", prefix + "/review", origin, media)
        literals(ret, 'case "edit", "image"', "return true", prefix + "/", "GET", media)
        if kind == "track":
            literals(ret, 'case "add-listen"', "return kind", prefix + "/", "GET", media)
        literals(admin, 'case "alias-add"', "return true", prefix + "/", "POST", definition("mediaAction", "internal/adminui/media.go"))
        reg, _ = rows["POST", prefix + "/confirm"]
        rows["POST", prefix + "/confirm"] = reg, definition("confirmCatalog", "internal/adminui/*.go")

    add("GET", "/", definition("IsPage", web), web_handler)
    add("GET", "/{slug}", definition("IsPage", web), web_handler)
    literals(web, 'case "listens"', "return true", "/{slug}/", "GET", web_handler)
    legacy = definition("RedirectLegacy", web)
    for kind in ("artist", "album", "track"):
        add("GET", "/{slug}/" + kind + "/{id}", loc(web, source(web).index("return len(parts) == 3")), web_handler)
        add("GET", "/" + kind + "/{id}", definition("safeNext", "internal/webui/handlers.go"), legacy)
    literals("internal/webui/handlers.go", 'if p == "listens"', "return raw", "/", "GET", legacy)


def extract_preview():
    file = "internal/webui/preview.go"
    go = source(file)
    for match in re.finditer(r'mux\.Handle(Func)?\("(GET )?([^"]+)"', go):
        method, path = "GET/HEAD", match[3]
        if path.endswith("/"):
            if path == "/preview-assets/":
                path += "*"
        reg = loc(file, match.start())
        handler = definition("Handler", "internal/webassets/http.go") if match[1] is None else reg
        add(method, path, reg, handler)
        if path == "/":
            add(method, "/{slug}", reg, handler)


def check_catalog():
    catalog = (Path(__file__).with_name("ROTAS.md")).read_text()
    listed = re.findall(r"^\| `([^`]+)` \| `([^`]+)` \|", catalog, re.M)
    missing = sorted(set(rows) - set(listed))
    extra = sorted(set(listed) - set(rows))
    if missing or extra or len(listed) != len(rows):
        raise SystemExit(f"Catálogo divergente: faltam {missing}; sobram {extra}; linhas={len(listed)}")
    print(f"{len(rows)} padrões no código e na tabela, sem duplicatas")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument("--preview", action="store_true", help="extrai somente cmd/ui-preview")
    mode.add_argument("--check", action="store_true", help="confere os padrões da tabela ROTAS.md")
    args = parser.parse_args()
    if args.preview:
        extract_preview()
    else:
        extract()
    if args.check:
        check_catalog()
    else:
        for key, references in sorted(rows.items()):
            print(*key, *references, sep="\t")
