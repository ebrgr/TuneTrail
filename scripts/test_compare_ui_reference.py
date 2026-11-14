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

if __name__ == '__main__':
    unittest.main()
