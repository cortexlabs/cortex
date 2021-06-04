from typing import DefaultDict
from fastapi import FastAPI
from pydantic import BaseModel

app = FastAPI()


def generate_primes(limit=None):
    """Sieve of Eratosthenes"""
    not_prime = DefaultDict(list)
    num = 2
    while limit is None or num <= limit:
        if num in not_prime:
            for prime in not_prime[num]:
                not_prime[prime + num].append(prime)
            del not_prime[num]
        else:
            yield num
            not_prime[num * num] = [num]
        num += 1


@app.get("/healthz")
def healthz():
    return "ok"


class Body(BaseModel):
    primes_to_generate: float


@app.post("/")
def prime_numbers(body: Body):
    return {"prime_numbers": list(generate_primes(body.primes_to_generate))}
