def clamp(value, low, high):
    if value < low:
        return low
    if value > high:
        return high
    return value


def is_even(n):
    return n % 2 == 0
