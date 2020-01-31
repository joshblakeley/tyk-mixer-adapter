FROM golang:1.13 as builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
COPY . /app
WORKDIR /app

# Copy and download dependency using go mod
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the code into the container
COPY . .

RUN go build -a -installsuffix cgo -v -o /app/tykgrpcadapter

FROM alpine:3.11
COPY --from=builder /app/tykgrpcadapter /app/
EXPOSE 5000
ENTRYPOINT [ "/app/tykgrpcadapter" ]
