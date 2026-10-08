#!/bin/sh
# Installs the Arch package of the latest main from GitHub (built by the
# Arch package workflow, .github/workflows/arch.yml), without building it
# here. Needs gh and sudo.
set -e
dir=$(mktemp -d)
trap 'rm -rf "$dir"' EXIT
gh release download main-latest -R LaPingvino/accolade -p 'accolade-git-r*.pkg.tar.zst' -D "$dir"
sudo pacman -U --noconfirm "$dir"/accolade-git-r*.pkg.tar.zst
