FROM golang:1.21-alpine

RUN apk add --no-cache gcc g++ libc-dev zeromq-dev zeromq git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build para bot sem a tag client
RUN go build -tags "!client" -o bot bot.go

# Build para cliente com a tag client  
RUN go build -tags client -o client main.go

CMD ["./client"]
