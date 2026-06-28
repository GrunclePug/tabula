# Build stage
FROM golang:alpine AS builder
WORKDIR /app
RUN apk add --no-cache curl make gcompat libgcc libstdc++
COPY . .
RUN CGO_ENABLED=0 make build

# Runtime stage
FROM gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=builder /app/bin/tabula /tabula
EXPOSE 8080
ENTRYPOINT ["/tabula"]
