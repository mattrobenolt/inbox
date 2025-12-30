#!/usr/bin/env sh
set -eu

repo="mattrobenolt/inbox"
binary="inbox"

os="$(uname -s)"
case "$os" in
Darwin) os="darwin" ;;
Linux) os="linux" ;;
*)
    echo "Unsupported OS: $os" >&2
    exit 1
    ;;
esac

arch="$(uname -m)"
case "$arch" in
x86_64) arch="amd64" ;;
arm64 | aarch64) arch="arm64" ;;
*)
    echo "Unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

version="${VERSION:-}"
if [ -z "$version" ]; then
    tag="$(curl -sSfL "https://api.github.com/repos/${repo}/releases/latest" | awk -F '\"' '/\"tag_name\":/ {print $4; exit}')"
    if [ -z "$tag" ]; then
        echo "Unable to determine latest version." >&2
        exit 1
    fi
    version="${tag#v}"
fi

asset="${binary}_${version}_${os}_${arch}.tar.gz"
url="https://github.com/${repo}/releases/download/v${version}/${asset}"
checksums_url="https://github.com/${repo}/releases/download/v${version}/checksums.txt"

tmpdir="$(mktemp -d)"
cleanup() { rm -rf "$tmpdir"; }
trap cleanup EXIT

curl -sSfL "$checksums_url" -o "$tmpdir/checksums.txt"
curl -sSfL "$url" -o "$tmpdir/$asset"
expected="$(awk -v file="$asset" '$2 == file {print $1; exit}' "$tmpdir/checksums.txt")"
if [ -z "$expected" ]; then
    echo "Checksum not found for $asset" >&2
    exit 1
fi

if command -v shasum >/dev/null 2>&1; then
    actual="$(shasum -a 256 "$tmpdir/$asset" | awk '{print $1}')"
elif command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "$tmpdir/$asset" | awk '{print $1}')"
else
    echo "Missing sha256 tool (shasum or sha256sum) for checksum verification" >&2
    exit 1
fi

if [ "$expected" != "$actual" ]; then
    echo "Checksum mismatch for $asset" >&2
    exit 1
fi
tar -xzf "$tmpdir/$asset" -C "$tmpdir"

bindir="${BIN_DIR:-}"
if [ -z "$bindir" ]; then
    if [ -w "/usr/local/bin" ]; then
        bindir="/usr/local/bin"
    else
        bindir="$HOME/.local/bin"
    fi
fi

mkdir -p "$bindir"
install -m 0755 "$tmpdir/$binary" "$bindir/$binary"
echo "Installed $binary to $bindir/$binary"
