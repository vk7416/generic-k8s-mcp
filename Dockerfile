FROM golang:1.23 AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/k8s-mcp-server ./cmd/k8s-mcp-server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/k8s-mcp-server /k8s-mcp-server
USER nonroot:nonroot
ENTRYPOINT ["/k8s-mcp-server"]
