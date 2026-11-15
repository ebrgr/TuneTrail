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
