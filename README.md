## Task

Design and implement "Word of Wisdom" tcp server:

- TCP server should be protected from DDOS attacks with the Prof of Work (https://en.wikipedia.org/wiki/Proof_of_work), the challenge-response protocol should be used.
- The choice of the POW algorithm should be explained.
- After Prof Of Work verification, server should send one of the quotes from “word of wisdom” book or any other collection of the quotes.
- Docker file should be provided both for the server and for the client that solves the POW challenge.

## Implementation details

The project is based on the idea of MVP, many patterns are omitted, and minimal interfaces are used on purpose.

## Solution 

The main idea is that the server generates a random challenge and sends it to the client. The challenge is represented as a JSON object that includes an array of one or more challenge strings, a difficulty value, and an algorithm used for hashing. The client solves the challenge and sends the solution back to the server. 

I opted for the SHA-256 Hashcash algorithm because of its simplicity and ease of implementation. While the client will need to expend considerable computational effort to generate a valid hash, verifying the result on the server side is quick and efficient. The client's task is to find a number that, when combined with the hash, results in a value that begins with a specified number of leading zeros. SHA-256 is widely recognized and performs well, making it ideal for scenarios where the server experiences high load. Compared to more resource-intensive algorithms like Scrypt, it is a lighter alternative, offering a good balance between security and resource usage. The difficulty can also be easily adjusted to meet the varying demands on the server.

## Getting started

Requirements:

- Go 1.22.5 
- Docker to run dockerfiles

```
# Run server 
make run-server

# Run client
make run-client
```

## Resources

- [Word of Wisdom](https://en.wikipedia.org/wiki/Word_of_Wisdom)
- [Proof of work](https://en.wikipedia.org/wiki/Proof_of_work)
- [Hashcash](https://en.wikipedia.org/wiki/Hashcash)
