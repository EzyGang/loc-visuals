#!/usr/bin/env sh
set -eu

repository="https://github.com/EzyGang/loc-visuals"
version="${LOC_VISUALS_VERSION:-latest}"
install_dir="${LOC_VISUALS_INSTALL_DIR:-${HOME}/.local/bin}"

for command in awk curl grep install mktemp tar uname; do
    if ! command -v "$command" >/dev/null 2>&1; then
        printf 'loc-visuals installer: required command not found: %s\n' "$command" >&2
        exit 1
    fi
done

case "$(uname -s)" in
    Linux) operating_system="linux" ;;
    Darwin) operating_system="darwin" ;;
    *)
        printf 'loc-visuals installer: unsupported operating system: %s\n' "$(uname -s)" >&2
        exit 1
        ;;
esac

case "$(uname -m)" in
    x86_64 | amd64) architecture="amd64" ;;
    arm64 | aarch64) architecture="arm64" ;;
    *)
        printf 'loc-visuals installer: unsupported architecture: %s\n' "$(uname -m)" >&2
        exit 1
        ;;
esac

if [ "$version" = "latest" ]; then
    release_url="$(curl -fsSL -o /dev/null -w '%{url_effective}' "$repository/releases/latest")"
    tag="${release_url##*/}"
    version="${tag#v}"
else
    version="${version#v}"
    tag="v${version}"
fi

if ! printf '%s\n' "$version" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'; then
    printf 'loc-visuals installer: invalid release version: %s\n' "$version" >&2
    exit 1
fi

archive="loc-visuals-${version}-${operating_system}-${architecture}.tar.gz"
download_root="$repository/releases/download/$tag"
temporary="$(mktemp -d)"
trap 'rm -rf "$temporary"' EXIT HUP INT TERM

curl -fsSL "$download_root/$archive" -o "$temporary/$archive"
curl -fsSL "$download_root/SHA256SUMS" -o "$temporary/SHA256SUMS"

expected="$(awk -v file="$archive" '$2 == file { print $1 }' "$temporary/SHA256SUMS")"
if [ -z "$expected" ]; then
    printf 'loc-visuals installer: checksum is missing for %s\n' "$archive" >&2
    exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "$temporary/$archive" | awk '{ print $1 }')"
elif command -v shasum >/dev/null 2>&1; then
    actual="$(shasum -a 256 "$temporary/$archive" | awk '{ print $1 }')"
else
    printf 'loc-visuals installer: sha256sum or shasum is required\n' >&2
    exit 1
fi

if [ "$actual" != "$expected" ]; then
    printf 'loc-visuals installer: checksum verification failed for %s\n' "$archive" >&2
    exit 1
fi

tar -xzf "$temporary/$archive" -C "$temporary"
mkdir -p "$install_dir"
install -m 0755 "$temporary/loc-visuals" "$install_dir/loc-visuals"
printf 'Installed loc-visuals %s to %s/loc-visuals\n' "$version" "$install_dir"

case ":${PATH}:" in
    *":${install_dir}:"*) ;;
    *) printf 'Add %s to PATH to run loc-visuals.\n' "$install_dir" ;;
esac
