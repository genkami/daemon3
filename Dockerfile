FROM golang:1.16 as builder

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

COPY pkg/ pkg/
COPY cmd/ cmd/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o daemon3 ./cmd/daemon3

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/daemon3 .
USER 65532:65532

ENTRYPOINT ["/daemon3"]
