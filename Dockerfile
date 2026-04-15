FROM golang:1.26.2-alpine3.23@sha256:c216c4343b489259302908b67a3c8fa55b283bdc30be729baa38b9953ca28857 AS build_deps

RUN apk add --no-cache git

WORKDIR /workspace

COPY go.mod .
COPY go.sum .

RUN go mod download

RUN go mod tidy

FROM build_deps AS build

COPY . .

RUN CGO_ENABLED=0 go build -o ionos-exporter -ldflags '-w -extldflags "-static"' .

FROM alpine:3.23@sha256:59855d3dceb3ae53991193bd03301e082b2a7faa56a514b03527ae0ec2ce3a95

RUN apk upgrade --no-cache
RUN apk add --no-cache ca-certificates

COPY --from=build /workspace/ionos-exporter /usr/local/bin/ionos-exporter

ENTRYPOINT ["ionos-exporter"]
