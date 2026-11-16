"""Exercise the web style limit on a disposable tree."""
import importlib.util
import os
import tempfile
import unittest
from pathlib import Path

spec = importlib.util.spec_from_file_location('check_web_style', Path(__file__).with_name('check_web_style.py'))
check = importlib.util.module_from_spec(spec)
spec.loader.exec_module(check)


class WebStyleTests(unittest.TestCase):
    def tree(self, html, css=''):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        root = Path(temp.name)
        (root / 'assets/templates/public').mkdir(parents=True)
        (root / 'assets/css').mkdir(parents=True)
        (root / 'assets/templates/public/a.html').write_text(html)
        (root / 'assets/css/a.css').write_text(css)
        return root

    def test_accepts_short_lines_and_single_long_attribute(self):
        url = 'href="https://example.test/' + 'x' * 200 + '"'
        self.assertEqual(list(check.offenders(self.tree('<a\n  ' + url + '\n>'))), [])

    def test_accepts_single_translation_string(self):
        line = '      "' + 'A long default translation. ' * 8 + '"}}'
        self.assertEqual(list(check.offenders(self.tree('{{tr "key"\n' + line + '\n'))), [])

    def test_rejects_long_template_and_css_lines(self):
        root = self.tree('<p>' + 'word ' * 40 + '</p>', '.a{' + 'color:red;' * 20 + '}')
        found = [(str(p), n) for p, n, _ in check.offenders(root)]
        self.assertEqual(found, [('assets/templates/public/a.html', 1), ('assets/css/a.css', 1)])

    def test_checks_nested_stylesheets(self):
        root = self.tree('')
        (root / 'assets/css/ui/pages').mkdir(parents=True)
        (root / 'assets/css/ui/pages/b.css').write_text('.b{' + 'color:red;' * 20 + '}')
        self.assertEqual([str(p) for p, _, _ in check.offenders(root)], ['assets/css/ui/pages/b.css'])

    def test_stylesheet_line_limit(self):
        root = self.tree('', 'a{}\n' * check.CSS_LINES)
        self.assertEqual(list(check.long_stylesheets(root)), [])
        (root / 'assets/css/a.css').write_text('a{}\n' * (check.CSS_LINES + 1))
        self.assertEqual([(str(p), n) for p, n in check.long_stylesheets(root)], [('assets/css/a.css', check.CSS_LINES + 1)])

    def test_viewport_breakpoint_policy(self):
        """Only the three canonical min-width queries may affect layout."""
        for condition in ('(min-width:40rem)', '(min-width: 64rem)', '(min-width:90rem)',
                          '(prefers-reduced-motion:reduce)', '(forced-colors:active)', '(hover:hover)'):
            with self.subTest(condition=condition):
                self.assertEqual(list(check.width_media_offenders(self.tree('', '@media '+condition+' {a{}}'))), [])
        for condition in ('(min-width:48rem)', '(max-width:640px)', '(width >= 40rem)',
                          '(min-width:640px)', '(min-width:40rem), (min-width:80rem)',
                          '(min-width:40rem) and (max-width:90rem)', '(min-height:50rem)',
                          '(orientation:landscape)', '(min-width:calc(40rem))'):
            with self.subTest(condition=condition):
                self.assertEqual(len(list(check.width_media_offenders(self.tree('', '@media '+condition+' {a{}}')))), 1)

    def test_media_comments_multiline_and_escapes(self):
        """Comments do not add queries or hide disallowed escaped feature names."""
        css = '/* @media(max-width:1px) {} */\n@media\n(min-width: /* shared */40rem) {a{content:"@media(max-width:2px)"}}'
        self.assertEqual(list(check.width_media_offenders(self.tree('', css))), [])
        css = r'@m\65 dia (min-w\69 dth:48rem) {a{}}'
        self.assertEqual(len(list(check.width_media_offenders(self.tree('', css)))), 1)

    def test_script_budget(self):
        root = self.tree('')
        (root / 'assets/js').mkdir(parents=True)
        (root / 'assets/js/search.js').write_bytes(os.urandom(8192))
        self.assertEqual([name for name, _, _ in check.over_budget(root)], ['assets/js/search.js'])
        (root / 'assets/js/search.js').write_text('const a = 1;')
        self.assertEqual(list(check.over_budget(root)), [])

    def test_rejects_long_attribute_with_spaces(self):
        self.assertEqual(len(list(check.offenders(self.tree('  title="' + 'a b ' * 50 + '"')))), 1)


if __name__ == '__main__':
    unittest.main()
