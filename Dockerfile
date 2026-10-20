FROM golang:1.27-trixie AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# The scratch base image as a blank canvas for prod stage
FROM scratch

WORKDIR /

# We copy the passwd file, essential for our non-root user
COPY --from=build /etc/passwd /etc/passwd

# This line will copy timezone info to final image
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo

# This line will copy all certificates to final image
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=build /out/api /api
EXPOSE 8080

ENTRYPOINT ["/api"]
