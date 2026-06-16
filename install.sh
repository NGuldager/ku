#!/bin/sh
set -e

REPO="bjarneo/ku"
NAME="ku"

need() {
    if ! command -v "$1" >/dev/null 2>&1; then
        echo "install: $1 is required" >&2
        exit 1
    fi
}

download() {
    url=$1
    dest=$2
    if command -v curl >/dev/null 2>&1; then
        curl -fSL -o "$dest" "$url"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO "$dest" "$url"
    else
        echo "install: curl or wget is required" >&2
        exit 1
    fi
}

choose_install_dir() {
    if [ -n "$INSTALL_DIR" ]; then
        printf '%s\n' "$INSTALL_DIR"
        return
    fi

    local_bin="$HOME/.local/bin"
    case ":$PATH:" in
        *":$local_bin:"*)
            mkdir -p "$local_bin"
            printf '%s\n' "$local_bin"
            return
            ;;
    esac

    if [ -d /usr/local/bin ]; then
        printf '%s\n' /usr/local/bin
        return
    fi

    last_path=$(printf '%s' "$PATH" | tr ':' '\n' | awk 'NF { last=$0 } END { print last }')
    if [ -n "$last_path" ]; then
        printf '%s\n' "$last_path"
        return
    fi

    echo "install: could not determine an install directory" >&2
    exit 1
}

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "install: unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

case "$OS" in
    linux|darwin) ;;
    mingw*|msys*|cygwin*) OS="windows" ;;
    *) echo "install: unsupported OS: $OS" >&2; exit 1 ;;
esac

ASSET="${NAME}-${OS}-${ARCH}"
TARGET_NAME="$NAME"
if [ "$OS" = "windows" ]; then
    ASSET="${ASSET}.exe"
    TARGET_NAME="${NAME}.exe"
fi

INSTALL_DIR=$(choose_install_dir)
URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"
CHECKSUM_URL="https://github.com/${REPO}/releases/latest/download/checksums.txt"

TMP=$(mktemp)
CHECKSUMS=$(mktemp)
cleanup() {
    rm -f "$TMP" "$CHECKSUMS"
}
trap cleanup EXIT INT TERM

echo "Downloading ${ASSET}..."
download "$URL" "$TMP"

echo "Verifying checksum..."
download "$CHECKSUM_URL" "$CHECKSUMS"
EXPECTED=$(awk -v asset="$ASSET" '$2 == asset { print $1 }' "$CHECKSUMS")
if [ -z "$EXPECTED" ]; then
    echo "install: checksum for ${ASSET} not found" >&2
    exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL=$(sha256sum "$TMP" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
    ACTUAL=$(shasum -a 256 "$TMP" | awk '{print $1}')
else
    echo "install: sha256sum or shasum is required" >&2
    exit 1
fi

if [ "$ACTUAL" != "$EXPECTED" ]; then
    echo "install: checksum mismatch" >&2
    echo "  expected: $EXPECTED" >&2
    echo "  got:      $ACTUAL" >&2
    exit 1
fi

chmod +x "$TMP"
if [ ! -d "$INSTALL_DIR" ]; then
    if ! mkdir -p "$INSTALL_DIR" 2>/dev/null; then
        need sudo
        sudo mkdir -p "$INSTALL_DIR"
    fi
fi
DEST="${INSTALL_DIR}/${TARGET_NAME}"

if [ -w "$INSTALL_DIR" ]; then
    install -m 0755 "$TMP" "$DEST"
else
    need sudo
    sudo install -m 0755 "$TMP" "$DEST"
fi

echo "Installed ${NAME} to ${DEST}"
case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *) echo "Add ${INSTALL_DIR} to PATH to run ${NAME} from anywhere." ;;
esac
