from utils import clamp, is_even


def test_clamp():
    assert clamp(5, 0, 10) == 5
    assert clamp(-1, 0, 10) == 0
    assert clamp(15, 0, 10) == 10


def test_is_even():
    assert is_even(4) is True
    assert is_even(3) is False
