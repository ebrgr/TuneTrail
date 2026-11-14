"""Comparator regressions: whitespace is advisory, missing and changed assets fail."""
import tempfile
import unittest
from pathlib import Path
try:
    from .compare_ui_reference import compare
except ImportError:
    from compare_ui_reference import compare

class ReferenceTests(unittest.TestCase):
    def test_comparison_modes(self):
        with tempfile.TemporaryDirectory() as tmp:
            a, b = Path(tmp)/'a', Path(tmp)/'b'
            a.mkdir(); b.mkdir()
            for left, right, code in [(b'<p>x</p>', b'<p>x</p>', 0),
                                      (b'<p>x</p>\n<p>y</p>', b'<p>x</p><p>y</p>', 1),
                                      (b'<span>x</span> <b>y</b>', b'<span>x</span><b>y</b>', 2),
                                      (b'x', b'y', 2)]:
                (a/'index.html').write_bytes(left); (b/'index.html').write_bytes(right)
                self.assertEqual(compare(a,b)['exit_code'],code)
            (b/'index.html').unlink()
            self.assertEqual(compare(a,b)['exit_code'],2)
            (b/'extra.css').write_text('body{}')
            self.assertEqual(len(compare(a,b)['differences']),2)
            (b/'font.woff2').write_bytes(b'font')
            self.assertEqual(len(compare(a,b)['differences']),3)

    def test_reviewed_html_overlay_preserves_asset_checks(self):
        """An HTML overlay cannot authorize asset changes or introduce new fixtures."""
        with tempfile.TemporaryDirectory() as tmp:
            base, candidate, overlay = [Path(tmp)/name for name in ('base', 'candidate', 'overlay')]
            for directory in (base, candidate, overlay):
                directory.mkdir()
            (base/'index.html').write_text('<p>original</p>')
            (candidate/'index.html').write_text('<p>reviewed</p>')
            (overlay/'index.html').write_text('<p>reviewed</p>')
            for directory in (base, candidate):
                (directory/'ui.css').write_text('body{}')
            self.assertEqual(compare(base, candidate, overlay)['exit_code'], 0)
            (candidate/'ui.css').write_text('changed')
            self.assertEqual(compare(base, candidate, overlay)['exit_code'], 2)
            (candidate/'index.html').write_text('<p>unreviewed</p>')
            self.assertEqual(compare(base, candidate, overlay)['exit_code'], 2)
            for name in ('unknown.html', 'ui.css'):
                with self.subTest(name=name):
                    extra = overlay/name
                    extra.write_text('unexpected')
                    with self.assertRaises(ValueError):
                        compare(base, candidate, overlay)
                    extra.unlink()

if __name__ == '__main__':
    unittest.main()
