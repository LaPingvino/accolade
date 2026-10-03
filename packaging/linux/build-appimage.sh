#!/bin/sh
# Builds Accolade-<version>-<arch>.AppImage from a built accolade binary.
#   packaging/linux/build-appimage.sh <binary> <version> [outdir]
set -eu
bin=$1
version=$2
out=${3:-dist}
arch=$(uname -m)
here=$(cd "$(dirname "$0")" && pwd)
root=$(cd "$here/../.." && pwd)

appdir=$(mktemp -d)/Accolade.AppDir
mkdir -p "$appdir/usr/bin" "$appdir/usr/share/applications" "$appdir/usr/share/icons/hicolor/512x512/apps"
install -m755 "$bin" "$appdir/usr/bin/accolade"
install -m644 "$here/accolade.desktop" "$appdir/usr/share/applications/accolade.desktop"
install -m644 "$root/Icon.png" "$appdir/usr/share/icons/hicolor/512x512/apps/accolade.png"
cp "$here/accolade.desktop" "$appdir/accolade.desktop"
cp "$root/Icon.png" "$appdir/accolade.png"
cat > "$appdir/AppRun" <<'RUN'
#!/bin/sh
here=$(dirname "$(readlink -f "$0")")
exec "$here/usr/bin/accolade" "$@"
RUN
chmod +x "$appdir/AppRun"

tool=${APPIMAGETOOL:-}
if [ -z "$tool" ]; then
	tool=$(mktemp -d)/appimagetool
	curl -fsSL -o "$tool" "https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-$arch.AppImage"
	chmod +x "$tool"
fi
mkdir -p "$out"
ARCH=$arch "$tool" --appimage-extract-and-run "$appdir" "$out/Accolade-$version-$arch.AppImage"
