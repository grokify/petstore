# syntax=docker/dockerfile:1
#
# petstore-server image:
#
#   docker build -t petstore-server .

FROM golang:1.27.1-alpine AS build
WORKDIR /src/grokify/petstore
COPY go.mod go.sum ./
RUN go mod download
COPY petstore.go openapi.yaml ./
COPY server ./server
COPY cmd ./cmd
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/petstore-server ./cmd/petstore-server

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/petstore-server /petstore-server
EXPOSE 8080
ENTRYPOINT ["/petstore-server"]
CMD ["--listen", ":8080"]
