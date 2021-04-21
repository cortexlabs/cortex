from collections import defaultdict


class Handler:
    def __init__(self, config, proto_module_pb2):
        self.proto_module_pb2 = proto_module_pb2

    def Predict(self, payload):
        prime_numbers_to_generate: int = payload.prime_numbers_to_generate
        for prime_number in self.gen_primes():
            if prime_numbers_to_generate == 0:
                break
            prime_numbers_to_generate -= 1
            yield self.proto_module_pb2.Output(prime_number=prime_number)

    def gen_primes(self, limit=None):
        """Sieve of Eratosthenes"""
        not_prime = defaultdict(list)
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
