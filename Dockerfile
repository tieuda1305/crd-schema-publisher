FROM --platform=$BUILDPLATFORM golang:1.26.2 AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /crd-schema-publisher ./cmd/

FROM gcr.io/distroless/static:nonroot
COPY --from=build /crd-schema-publisher /crd-schema-publisher
ENTRYPOINT ["/crd-schema-publisher"]
