#!/usr/bin/env bash
# Install development dependencies on the host (Arch, Debian, Fedora).
# Usage: deps.sh <go_version>
set -euo pipefail

GO_VERSION="${1:?Usage: deps.sh <go_version>}"

# ── Detect distro family ─────────────────────────────────────────────────────

if [ ! -f /etc/os-release ]; then
  echo "ERROR: Cannot detect distribution (missing /etc/os-release)" >&2
  exit 1
fi
# shellcheck source=/dev/null
. /etc/os-release

detect_family() {
  case "${ID:-}" in
    arch|endeavouros|manjaro|cachyos|garuda|artix) echo "arch" ;;
    fedora|centos|rhel|rocky|alma|nobara)          echo "fedora" ;;
    debian|ubuntu|linuxmint|pop|elementary|zorin)   echo "debian" ;;
    *)
      case "${ID_LIKE:-}" in
        *arch*)           echo "arch" ;;
        *fedora*|*rhel*)  echo "fedora" ;;
        *debian*|*ubuntu*) echo "debian" ;;
        *)                echo "unknown" ;;
      esac ;;
  esac
}

FAMILY=$(detect_family)
echo "==> Detected distro: ${PRETTY_NAME:-$ID} (family: $FAMILY)"

# ── Install system packages ──────────────────────────────────────────────────

case "$FAMILY" in
  debian)
    echo "==> Installing system packages via apt..."
    sudo apt-get update
    sudo apt-get install -y --no-install-recommends \
      aptly build-essential ca-certificates curl debhelper dpkg-dev \
      git gnupg just lintian \
      python3 python3-pip python3-setuptools python3-venv python3-wheel \
      pipx
    ;;
  arch)
    echo "==> Installing system packages via pacman..."
    sudo pacman -Sy --needed --noconfirm \
      base-devel ca-certificates curl git gnupg dpkg just \
      python python-pip python-setuptools python-wheel python-pipx
    echo ""
    echo "NOTE: Debian packaging tools (aptly, debhelper, lintian) are"
    echo "      AUR-only. Install them separately if you need build/publish."
    ;;
  fedora)
    echo "==> Installing system packages via dnf..."
    sudo dnf install -y \
      gcc gcc-c++ make ca-certificates curl git gnupg2 just \
      python3 python3-pip python3-setuptools python3-wheel pipx
    echo ""
    echo "NOTE: Debian packaging tools (aptly, debhelper, lintian) are not"
    echo "      available via dnf. Only needed for build/publish commands."
    ;;
  *)
    echo "ERROR: Unsupported distribution '${ID:-unknown}'." >&2
    echo "Supported families: Arch-based, Debian-based, Fedora-based." >&2
    exit 1
    ;;
esac

# ── Install or verify Go toolchain ───────────────────────────────────────────

GO_REQUIRED="${GO_VERSION%.*}"  # e.g. 1.25.0 → 1.25
CURRENT_GO=""
if command -v go &>/dev/null; then
  CURRENT_GO="$(go version | awk '{gsub(/^go/, "", $3); print $3}')"
fi

if [[ -z "$CURRENT_GO" || ! "$CURRENT_GO" =~ ^${GO_REQUIRED} ]]; then
  echo "==> Installing Go ${GO_VERSION} (current: ${CURRENT_GO:-none})..."
  ARCH="$(uname -m)"
  case "$ARCH" in
    x86_64)          GOARCH="amd64" ;;
    aarch64|arm64)   GOARCH="arm64" ;;
    *) echo "ERROR: Unsupported architecture: $ARCH" >&2; exit 1 ;;
  esac
  TMP="$(mktemp)"
  curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${GOARCH}.tar.gz" -o "$TMP"
  sudo rm -rf /usr/local/go
  sudo tar -C /usr/local -xzf "$TMP"
  rm -f "$TMP"
  export PATH="/usr/local/go/bin:$PATH"
  echo "==> Installed: $(go version)"
  if ! echo "$PATH" | tr ':' '\n' | grep -qx '/usr/local/go/bin'; then
    echo "    Add to your shell profile: export PATH=\"/usr/local/go/bin:\$PATH\""
  fi
else
  echo "==> Go ${CURRENT_GO} satisfies requirement (>= ${GO_REQUIRED})."
fi

# ── Install pipx packages ───────────────────────────────────────────────────

echo "==> Installing uv and pip-tools via pipx..."
pipx ensurepath 2>/dev/null || true
pipx install uv 2>/dev/null || pipx upgrade uv 2>/dev/null || true
pipx install pip-tools 2>/dev/null || pipx upgrade pip-tools 2>/dev/null || true

echo ""
echo "==> All dependencies installed."
echo "    Run 'just build' to compile, or 'just dev' for a devcontainer."
