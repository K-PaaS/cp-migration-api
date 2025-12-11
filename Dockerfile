FROM golang:alpine AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /build

COPY go.mod go.sum main.go ./

COPY api ./api
COPY docs ./docs
COPY model ./model
COPY config ./config

RUN go mod download

RUN go mod tidy

RUN go build -o main .

WORKDIR /dist

RUN cp /build/main .


COPY config.env .

FROM scratch

COPY --from=builder /dist/main .

COPY config.env .

ENV HMAC_KEY=${MIG_HMAC_KEY} \
    PRIVATE_KEY=${MIG_PRIVATE_KEY} \
    IS_ENCRYPTION=${IS_ENCRYPTION}

ENTRYPOINT ["/main"]

