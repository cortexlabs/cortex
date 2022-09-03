from typing import List
from random import sample

RANGE = 10 ** 12
LENGTH = 5


def generate_sample() -> List[int]:
    return sample(range(RANGE), LENGTH)
