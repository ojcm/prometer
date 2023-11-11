FROM golang:1.21

WORKDIR /app

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download # Improve build performance.
COPY internal internal
COPY *.go .

RUN go build .

ENTRYPOINT ["./prometer"]