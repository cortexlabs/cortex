FROM golang:1.17.3

COPY app /app

ENTRYPOINT ["go", "run", "/app/main.go"]
