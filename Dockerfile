# Build Stage
FROM golang:alpine AS builder
WORKDIR /app

# Kopiere die main.go
COPY main.go ./

# 1. Initialisiere das Modul
RUN go mod init t-cloud-cert-sync

# 2. Pinne Kubernetes-Pakete auf eine STABILE Version (v0.30.0), 
# um das Herunterladen von fehlerhaften Zukunfts-Tags zu verhindern!
RUN go get k8s.io/client-go@v0.30.0 \
           k8s.io/api@v0.30.0 \
           k8s.io/apimachinery@v0.30.0

# 3. Lade alle verbleibenden Abhängigkeiten (OTC SDK etc.)
RUN go mod tidy

# 4. Baue die Anwendung
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o t-cloud-cert-sync .

# Run Stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /app/t-cloud-cert-sync .

ENTRYPOINT ["./t-cloud-cert-sync"]
