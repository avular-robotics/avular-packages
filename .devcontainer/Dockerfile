ARG VARIANT=24.04
ARG BASE_IMAGE=ubuntu:${VARIANT}
ARG GO_VERSION=1.25.0
ARG USER_UID=1000
ARG USER_GID=1000
FROM ${BASE_IMAGE}
ARG GO_VERSION

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    aptly \
    build-essential \
    ca-certificates \
    curl \
    debhelper \
    dpkg-dev \
    git \
    gnupg \
    lintian \
    python3 \
    python3-pip \
    python3-setuptools \
    python3-venv \
    python3-wheel \
    just \
    pipx \
    sudo \
    && rm -rf /var/lib/apt/lists/*

ENV PIPX_HOME=/opt/pipx
ENV PIPX_BIN_DIR=/usr/local/bin
RUN pipx install uv && pipx install pip-tools

ENV PATH="/usr/local/go/bin:${PATH}"

RUN set -eux; \
    GO_VERSION="${GO_VERSION}"; \
    ARCH="$(uname -m)"; \
    case "$ARCH" in \
        x86_64) GOARCH="amd64" ;; \
        aarch64|arm64) GOARCH="arm64" ;; \
        *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;; \
    esac; \
    curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${GOARCH}.tar.gz" -o /tmp/go.tgz; \
    rm -rf /usr/local/go; \
    tar -C /usr/local -xzf /tmp/go.tgz; \
    rm /tmp/go.tgz; \
    /usr/local/go/bin/go version

RUN set -eux; \
    USER_UID="${USER_UID:-1000}"; \
    USER_GID="${USER_GID:-1000}"; \
    if ! getent group "${USER_GID}" >/dev/null; then groupadd -g "${USER_GID}" vscode; fi; \
    if id -u "${USER_UID}" >/dev/null 2>&1; then \
        EXISTING_USER="$(getent passwd "${USER_UID}" | cut -d: -f1)"; \
        if [ "${EXISTING_USER}" != "vscode" ]; then usermod -l vscode -d /home/vscode -m "${EXISTING_USER}"; fi; \
        usermod -g "${USER_GID}" vscode; \
    else \
        useradd -m -s /bin/bash -u "${USER_UID}" -g "${USER_GID}" vscode; \
    fi
RUN echo "vscode ALL=(ALL) NOPASSWD:ALL" >/etc/sudoers.d/vscode && chmod 0440 /etc/sudoers.d/vscode

USER vscode
WORKDIR /workspaces/avular-packages

