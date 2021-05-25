import os
from typing import DefaultDict

from fastapi import FastAPI
from pydantic import BaseModel


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


class Request(BaseModel):
    primes_to_generate: float


app = FastAPI()
record_only = os.getenv("RECORD_ONLY", "false")


@app.get("/healthz")
def healthz():
    return "ok"


@app.post("/")
def prime_numbers(request: Request):
    if record_only == "true":
        print(f"received payload", request.primes_to_generate)
        return

    return {"prime_numbers": list(generate_primes(request.primes_to_generate))}
