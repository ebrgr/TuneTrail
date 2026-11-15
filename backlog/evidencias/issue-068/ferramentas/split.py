"""Split assets/css/ui.css by function into layered files, keeping each file's rules in source order."""
import re, sys, collections
sys.path.insert(0, __import__('os').path.dirname(__file__))
from cssparse import chunks

ROOT = __import__('subprocess').run(['git', '-C', __import__('os').path.dirname(__import__('os').path.abspath(__file__)), 'rev-parse', '--show-toplevel'], capture_output=True, text=True, check=True).stdout.strip() + '/assets/css/'
# (file, layer) in bundle order; layer order matches the @layer statement.
FILES = [
    ('ui/reset.css', 'reset'), ('ui/tokens.css', 'tokens'), ('ui/base.css', 'base'),
    ('ui/layout/shell.css', 'layout'),
    ('ui/components/profile.css', 'components'), ('ui/components/section.css', 'components'),
    ('ui/components/media-list.css', 'components'), ('ui/components/activity.css', 'components'),
    ('ui/components/dialog.css', 'components'), ('ui/components/theme-panel.css', 'components'),
    ('ui/components/search.css', 'components'), ('ui/components/forms.css', 'components'),
    ('ui/pages/home.css', 'pages'), ('ui/pages/profiles.css', 'pages'), ('ui/pages/listing.css', 'pages'),
    ('ui/pages/media.css', 'pages'), ('ui/pages/rewind.css', 'pages'),
    ('ui/utilities.css', 'utilities'),
]
RULES = [
    # The navigation avatar overrides .avatar by specificity, so it must share its layer.
    (r'^\.nav-button \.avatar', 'ui/components/profile.css'),
    (r'^\.old-week', 'ui/components/activity.css'),
    (r'^h2\.search-section-title', 'ui/components/search.css'),
    (r'^\[hidden\]$|^\.sr-only$', 'ui/utilities.css'),
    (r'^\*,::before,::after$|^html$|^body,h1|^button,input|^button$|^button:disabled$|^a$|^img,svg$|^img$|^ol,ul$', 'ui/reset.css'),
    (r'^:root$', 'ui/tokens.css'),
    (r'^html,body$|^body$|^h[1-5]\b|^a,div$|^a:hover|^:is\(a,button|^\.(primary|secondary|error|card|offset)$|^summary$|^\.noscript$|^textarea$'
     r'|^body:has\(dialog|^\.cover-failed$|^select:|^button\.large-button', 'ui/base.css'),
    (r'^\.app-|^\.sidebar|^html\[data-nav-layout|^\.nav-|^body:has\(#nav-admin|^footer|^\.preview-tools', 'ui/layout/shell.css'),
    (r'^\.avatar|^\.profile-header|^\.admin-avatar|^\.admin-badge|^\[data-theme="pearl"\] \.admin-badge|^\.profile-identity', 'ui/components/profile.css'),
    (r'^\.section-|^\.period-selector', 'ui/components/section.css'),
    (r'^\.media(-text|-title|-meta|-artist)?\b(?!-)|^\.media-meta|^\.feature-list|^\.art-button|^\.medium|^\.small|^\.media img|^\.empty$|^\.tracks$'
     r'|^\.ranked-list|^\.rank\b|^\.plays$|^\.last-played$|^\.listens|^\.now-dot|^\.old-week', 'ui/components/media-list.css'),
    (r'^\.activity|^\.day-labels|^\.legend', 'ui/components/activity.css'),
    (r'^dialog|^\.close-dialog|^\.profile-dialog|^\.profile-options|^\.all-profiles|^@keyframes dialog-in', 'ui/components/dialog.css'),
    (r'^\.theme-|^\.panel-section|^\.nav-layout|^\.custom-theme|^\.reset-theme|^\.swatches', 'ui/components/theme-panel.css'),
    (r'^\.search-(?!page)|^h2\.search-section-title|^\[data-instant-search\]|^\[role=option\]', 'ui/components/search.css'),
    (r'^\.language-picker', 'ui/components/forms.css'),
    (r'^\.home|^\.overview|^\.stats|^\.feature-|^\.hero', 'ui/pages/home.css'),
    (r'^\.profile-selection|^\.eyebrow|^\.intro$|^\.profile-grid|^\.profile-card|^\.text-link|^\.admin-entry', 'ui/pages/profiles.css'),
    (r'^\.listing|^\.date-range|^\.pagination|^\.search-page-form|^\.see-more|^\.delete-listen|^\.listen-manage', 'ui/pages/listing.css'),
    (r'^\.media-|^\.interest-|^\.graph-', 'ui/pages/media.css'),
    (r'^\.rewind-', 'ui/pages/rewind.css'),
]
# Rules whose selector needs context: global reduced-motion goes to base.
MEDIA_FILE = {'@media(prefers-reduced-motion:reduce)': 'ui/base.css', '@media (forced-colors: active)': 'ui/base.css'}


DESC = {
    'ui/reset.css': 'Reset: browser defaults normalised like the original Tailwind preflight.',
    'ui/tokens.css': 'Tokens: layout custom properties; theme colours come from themes.css.',
    'ui/base.css': 'Base: element typography, focus, text colours, cards and native controls.',
    'ui/layout/shell.css': 'Layout: application shell, sidebars, navigation and footer.',
    'ui/components/profile.css': 'Components: avatar, profile header, identity and administrator badge.',
    'ui/components/section.css': 'Components: section titles, section links and the period selector.',
    'ui/components/media-list.css': 'Components: media items, ranked lists and listen tables.',
    'ui/components/activity.css': 'Components: activity heatmap and its legend.',
    'ui/components/dialog.css': 'Components: native dialogs and the profile switcher.',
    'ui/components/theme-panel.css': 'Components: theme panel popover, swatches and custom theme form.',
    'ui/components/search.css': 'Components: instant search dialog and result cards.',
    'ui/components/forms.css': 'Components: language picker form.',
    'ui/pages/home.css': 'Pages: home overview, hero and highlight blocks.',
    'ui/pages/profiles.css': 'Pages: profile selection.',
    'ui/pages/listing.css': 'Pages: listings, filters, pagination and the search page.',
    'ui/pages/media.css': 'Pages: artist, album and track pages with their graphs.',
    'ui/pages/rewind.css': 'Pages: rewind.',
    'ui/utilities.css': 'Utilities: always win over components (hidden, screen-reader only).',
}


def target(selector):
    for pattern, name in RULES:
        if re.search(pattern, selector):
            return name
    raise SystemExit(f'unmapped selector: {selector!r}')


def main():
    text = __import__('subprocess').run(['git', '-C', ROOT, 'show', 'cfd6717:assets/css/ui.css'], capture_output=True, text=True, check=True).stdout
    out = collections.OrderedDict((f, []) for f, _ in FILES)
    for raw, prelude, body, line in chunks(text):
        if prelude is None:
            continue  # trailing whitespace only
        if prelude.startswith('@media'):
            head = raw[:raw.index('{') + 1]
            lead, header = re.match(r'(\s*(?:/\*.*?\*/\s*)*)(.*)', head, re.S).groups()
            groups = []
            for iraw, ipre, _, _ in chunks(body):
                if ipre is None:
                    continue
                name = MEDIA_FILE.get(prelude) or target(ipre)
                if groups and groups[-1][0] == name:
                    groups[-1][1].append(iraw)
                else:
                    groups.append((name, [iraw]))
            for k, (name, parts) in enumerate(groups):
                one_line = '\n' not in raw.strip()
                block = (lead if k == 0 else '\n') + header + ''.join(parts) + ('\n}' if not one_line else ' }')
                out[name].append((line, block))
        else:
            out[target(prelude)].append((line, raw))
    layers = dict(FILES)
    for name, parts in out.items():
        content = ''.join(raw for _, raw in sorted(parts)).strip('\n')
        layer = layers[name]
        head = f'/* {DESC[name]} */\n'
        if name == 'ui/reset.css':
            head += '/* Cascade order for the whole bundle; later layers win regardless of specificity. */\n'
            head += '@layer reset, tokens, base, layout, components, pages, utilities;\n'
        content = f'{head}@layer {layer} {{\n{content}\n}}\n'
        open(ROOT + name, 'w').write(content)
        print(f'{name:40} {layers[name]:11} {content.count(chr(10)) + 1:5} lines  {len(parts)} chunks')


main()
