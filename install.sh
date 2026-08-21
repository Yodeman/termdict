#!/bin/sh
# TermDict installer for Linux and macOS.
#
# Usage:
#   ./install.sh                     install latest release to ~/.local/bin
#   ./install.sh --version v0.2.0    install a specific release
#   ./install.sh --system            install to /usr/local/bin (sudo)
#   ./install.sh --prefix DIR        install to DIR
#   ./install.sh --from-dir DIR      dev: install archives from a local
#                                    directory (e.g. goreleaser dist/) instead
#                                    of downloading
#   ./install.sh uninstall           remove the binary from the prefix
#
# The script never edits shell rc files; it prints PATH instructions.
set -eu

REPO="yodeman/termdict"
VERSION="latest"
FROM_DIR=""
PREFIX="${PREFIX:-${HOME}/.local/bin}"
SYSTEM=0

usage() { sed -n '2,16p' "$0"; exit 0; }

while [ $# -gt 0 ]; do
    case "$1" in
        --version)  [ $# -ge 2 ] || { echo "error: $1 needs a value" >&2; exit 2; }
                    VERSION="$2"; shift 2 ;;
        --prefix)   [ $# -ge 2 ] || { echo "error: $1 needs a value" >&2; exit 2; }
                    PREFIX="$2"; shift 2 ;;
        --system)   SYSTEM=1; shift ;;
        --from-dir) [ $# -ge 2 ] || { echo "error: $1 needs a value" >&2; exit 2; }
                    FROM_DIR="$2"; shift 2 ;;
        -h|--help)  usage ;;
        uninstall)  UNINSTALL=1; shift ;;
        *)          echo "error: unknown argument '$1'" >&2; exit 2 ;;
    esac
done

if [ "${SYSTEM}" = 1 ]; then
    PREFIX="/usr/local/bin"
fi

if [ "${UNINSTALL:-0}" = 1 ]; then
    if [ -f "${PREFIX}/termdict" ]; then
        rm -f "${PREFIX}/termdict"
        echo "Removed ${PREFIX}/termdict."
        echo "Your dictionary data was kept (see 'termdict help' for its location)."
    else
        echo "No termdict binary found at ${PREFIX}/termdict."
    fi
    exit 0
fi

# --- platform detection -------------------------------------------------
os=$(uname -s)
arch=$(uname -m)
case "${os}" in
    Linux)  os_name="linux"  ;;
    Darwin) os_name="darwin" ;;
    *) echo "error: unsupported OS '${os}' (Windows: use install.ps1)" >&2; exit 2 ;;
esac
case "${arch}" in
    x86_64|amd64)  arch_name="amd64" ;;
    aarch64|arm64) arch_name="arm64" ;;
    *) echo "error: unsupported architecture '${arch}'" >&2; exit 2 ;;
esac

workdir=$(mktemp -d /tmp/termdict-install.XXXXXX)
trap 'rm -rf "${workdir}"' EXIT

# --- resolve version ----------------------------------------------------
fetch() { # fetch <url> <dest>
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL -o "$2" "$1"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO "$2" "$1"
    else
        echo "error: need curl or wget to download releases" >&2; exit 3
    fi
}

if [ -n "${FROM_DIR}" ]; then
    archive_name="termdict_v${VERSION}_${os_name}_${arch_name}.tar.gz"
    [ -f "${FROM_DIR}/${archive_name}" ] || \
        archive_name="termdict_${os_name}_${arch_name}.tar.gz" # snapshot builds drop the tag
    if [ ! -f "${FROM_DIR}/${archive_name}" ]; then
        echo "error: no archive for ${os_name}/${arch_name} in ${FROM_DIR}" >&2; exit 3
    fi
    cp "${FROM_DIR}/${archive_name}" "${workdir}/archive.tar.gz"
else
    if [ "${VERSION}" = "latest" ]; then
        tag=$(fetch "https://api.github.com/repos/${REPO}/releases/latest" - | \
            sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n1)
        [ -n "${tag}" ] || { echo "error: could not resolve latest release" >&2; exit 3; }
    else
        tag="${VERSION}"
        case "${tag}" in v*) ;; *) tag="v${tag}" ;; esac
    fi
    echo "Installing termdict ${tag} (${os_name}/${arch_name})..."
    base="https://github.com/${REPO}/releases/download/${tag}"
    fetch "${base}/termdict_${tag}_${os_name}_${arch_name}.tar.gz" "${workdir}/archive.tar.gz"
    fetch "${base}/termdict_checksums.txt" "${workdir}/checksums.txt" || \
        echo "warning: checksums not available for this release; skipping verification"

    if [ -f "${workdir}/checksums.txt" ]; then
        want=$(grep "termdict_${tag}_${os_name}_${arch_name}.tar.gz" "${workdir}/checksums.txt" | awk '{print $1}')
        got=$(sha256sum "${workdir}/archive.tar.gz" 2>/dev/null | awk '{print $1}' ||
              shasum -a 256 "${workdir}/archive.tar.gz" | awk '{print $1}')
        if [ -n "${want}" ] && [ "${want}" != "${got}" ]; then
            echo "error: checksum mismatch for downloaded archive" >&2; exit 3
        fi
        echo "Checksum verified."
    fi
fi

tar -xzf "${workdir}/archive.tar.gz" -C "${workdir}"
[ -f "${workdir}/termdict" ] || { echo "error: archive did not contain a termdict binary" >&2; exit 3; }

# --- install ------------------------------------------------------------
do_privileged() {
    if [ -w "$(dirname "${PREFIX}")" ] || [ -w "${PREFIX}" ] || [ -w "/" ]; then
        mkdir -p "${PREFIX}"
        cp "${workdir}/termdict" "${PREFIX}/termdict"
        chmod 0755 "${PREFIX}/termdict"
    else
        sudo mkdir -p "${PREFIX}"
        sudo cp "${workdir}/termdict" "${PREFIX}/termdict"
        sudo chmod 0755 "${PREFIX}/termdict"
    fi
}
do_privileged

"${PREFIX}/termdict" --version && echo "Installed to ${PREFIX}/termdict."

case ":${PATH}:" in
    *":${PREFIX}:"*) ;;
    *)
        cat <<EOF

NOTE: ${PREFIX} is not in your PATH. Add it with:

    export PATH="\$PATH:${PREFIX}"

Consider adding that line to your ~/.profile, ~/.bashrc or ~/.zshrc.
EOF
        ;;
esac
