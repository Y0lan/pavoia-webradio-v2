"""
Unit tests for tag_extract normalization — runs without mutagen installed.

These exercise the normalization rules only. Full integration against a real
MP3 requires mutagen; that test file lives with the backfill script and is
run manually during initial deploy.
"""
from __future__ import annotations

import unittest
from tag_extract import (
    _normalize_bpm,
    _normalize_camelot,
    _normalize_year,
    _normalize_genre,
)


class BPMNormalizationTest(unittest.TestCase):
    def test_plain_int(self):
        self.assertEqual(_normalize_bpm("120"), 120)

    def test_decimal_rounds_up(self):
        self.assertEqual(_normalize_bpm("120.5"), 121)

    def test_decimal_rounds_down(self):
        self.assertEqual(_normalize_bpm("120.4"), 120)

    def test_multi_value_picks_first_valid(self):
        self.assertEqual(_normalize_bpm("120;123"), 120)

    def test_multi_value_skips_bad_first(self):
        self.assertEqual(_normalize_bpm("999;128"), 128)

    def test_below_range(self):
        self.assertIsNone(_normalize_bpm("30"))

    def test_above_range(self):
        self.assertIsNone(_normalize_bpm("300"))

    def test_malformed(self):
        self.assertIsNone(_normalize_bpm("fast"))

    def test_empty(self):
        self.assertIsNone(_normalize_bpm(""))
        self.assertIsNone(_normalize_bpm(None))

    def test_numeric_type(self):
        self.assertEqual(_normalize_bpm(128), 128)
        self.assertEqual(_normalize_bpm(127.6), 128)


class CamelotNormalizationTest(unittest.TestCase):
    def test_leading_zero_stripped(self):
        self.assertEqual(_normalize_camelot("05A"), "5A")

    def test_already_canonical(self):
        self.assertEqual(_normalize_camelot("5A"), "5A")

    def test_twelve(self):
        self.assertEqual(_normalize_camelot("12B"), "12B")
        self.assertEqual(_normalize_camelot("012B"), "12B")

    def test_lowercase_letter(self):
        self.assertEqual(_normalize_camelot("7a"), "7A")

    def test_whitespace(self):
        self.assertEqual(_normalize_camelot(" 08A "), "8A")

    def test_musical_key_rejected(self):
        self.assertIsNone(_normalize_camelot("Am"))
        self.assertIsNone(_normalize_camelot("C"))
        self.assertIsNone(_normalize_camelot("F#"))

    def test_out_of_range_letter(self):
        self.assertIsNone(_normalize_camelot("5C"))

    def test_out_of_range_number(self):
        self.assertIsNone(_normalize_camelot("13A"))
        self.assertIsNone(_normalize_camelot("0A"))

    def test_empty(self):
        self.assertIsNone(_normalize_camelot(""))
        self.assertIsNone(_normalize_camelot(None))


class YearNormalizationTest(unittest.TestCase):
    def test_year_only(self):
        self.assertEqual(_normalize_year("2021"), 2021)

    def test_full_date(self):
        self.assertEqual(_normalize_year("2021-05-14"), 2021)

    def test_int_type(self):
        self.assertEqual(_normalize_year(2019), 2019)

    def test_too_old(self):
        self.assertIsNone(_normalize_year("1899"))

    def test_future(self):
        # Current year + 1 is allowed (pre-release); +2 rejected.
        from datetime import datetime
        this_year = datetime.now().year
        self.assertEqual(_normalize_year(str(this_year + 1)), this_year + 1)
        self.assertIsNone(_normalize_year(str(this_year + 5)))

    def test_malformed(self):
        self.assertIsNone(_normalize_year("unknown"))
        self.assertIsNone(_normalize_year(""))
        self.assertIsNone(_normalize_year(None))


class GenreNormalizationTest(unittest.TestCase):
    def test_basic(self):
        self.assertEqual(_normalize_genre("Techno"), "Techno")

    def test_trimmed(self):
        self.assertEqual(_normalize_genre("  Melodic House & Techno  "), "Melodic House & Techno")

    def test_null_bytes_stripped(self):
        self.assertEqual(_normalize_genre("Deep House\x00\x00"), "Deep House")

    def test_empty(self):
        self.assertIsNone(_normalize_genre(""))
        self.assertIsNone(_normalize_genre("   "))
        self.assertIsNone(_normalize_genre(None))

    def test_preserves_case(self):
        # We deliberately don't canonicalize case — "DnB" and "dnb" both pass
        # through as-is. Taxonomy normalization is a separate pass.
        self.assertEqual(_normalize_genre("DnB"), "DnB")


if __name__ == "__main__":
    unittest.main()
