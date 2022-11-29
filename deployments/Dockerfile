# Compile
FROM golang:1.18 as builder

WORKDIR /follow.markets

COPY go.mod ./
RUN go mod download

COPY . .
RUN rm -r configs
RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false ./cmd/app

# This image is to deploy the binary
FROM alpine
RUN apk update
RUN apk add --update ca-certificates tzdata
COPY --from=builder ./follow.markets/app app
COPY configs/configs.json configs/configs.json
COPY configs/signals/signal.json configs/signals/signal.json
COPY configs/results/README.md configs/results/README.md
CMD ["./app"]
