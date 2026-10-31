# syntax=docker/dockerfile:1
#
# GoLock é um app de DESKTOP (Wails). O Docker aqui serve como "fábrica":
# ele compila o app dentro de um ambiente limpo e devolve os executáveis.
#
# Uso:  docker build --target export --output ./dist .
# Saída: ./dist/golock.exe (Windows) e ./dist/golock (Linux)

# ---------- Etapa 1: ambiente de compilação ----------
FROM golang:1.25-bookworm AS builder

ENV DEBIAN_FRONTEND=noninteractive

# A imagem oficial usa GOTOOLCHAIN=local e recusaria o go.mod (que pede Go mais novo
# que o da imagem). Com "auto" o Go baixa a versão exata que o go.mod exige.
ENV GOTOOLCHAIN=auto

# Dependências do Wails no Linux (GTK + WebKit) e Node.js para o frontend
RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.0-dev \
      curl ca-certificates gnupg \
 && curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
 && apt-get install -y --no-install-recommends nodejs \
 && rm -rf /var/lib/apt/lists/*

# CLI do Wails (mesma versão do go.mod)
RUN go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0

WORKDIR /app

# Copiar só os arquivos de dependências primeiro permite ao Docker
# reaproveitar o cache quando você muda apenas o código.
COPY go.mod go.sum ./
RUN go mod download

COPY frontend/package.json frontend/package-lock.json frontend/
RUN cd frontend && npm ci

# Agora o restante do projeto
COPY . .

# Compila para Windows e para Linux
RUN wails build -platform windows/amd64 -clean \
 && wails build -platform linux/amd64

# ---------- Etapa 2: só os resultados ----------
FROM scratch AS export
COPY --from=builder /app/build/bin/ /