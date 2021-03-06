#!/bin/bash

set -e

cd "$(dirname "$0")"

RUFS_VERSION="$1"
if [ -z "$RUFS_VERSION" ]; then
	RUFS_VERSION=$(git describe --tags --dirty --always | awk -F- '{print $1}' | cut -b 2- -)
fi

# Clean up
if [ -d "rufs.app" ]; then
	rm -rf rufs.app
fi
if [ -f "rufs-setup.pkg" ]; then
	rm rufs-setup.pkg
fi
if [ -d "rufs.iconset" ]; then
	rm -rf rufs.iconset
fi
if [ -d "tempdir" ]; then
	rm -rf tempdir
fi

# Create icons
mkdir -p rufs.iconset
sips -z 16 16 rufs-macos.png --out rufs.iconset/icon_16x16.png
sips -z 32 32 rufs-macos.png --out rufs.iconset/icon_16x16@2x.png
sips -z 32 32 rufs-macos.png --out rufs.iconset/icon_32x32.png
sips -z 64 64 rufs-macos.png --out rufs.iconset/icon_32x32@2x.png
sips -z 128 128 rufs-macos.png --out rufs.iconset/icon_128x128.png
sips -z 256 256 rufs-macos.png --out rufs.iconset/icon_128x128@2x.png
sips -z 256 256 rufs-macos.png --out rufs.iconset/icon_256x256.png
sips -z 512 512 rufs-macos.png --out rufs.iconset/icon_256x256@2x.png
sips -z 512 512 rufs-macos.png --out rufs.iconset/icon_512x512.png
cp rufs-macos.png rufs.iconset/icon_512x512@2x.png
iconutil -c icns -o rufs.icns rufs.iconset
rm -rf rufs.iconset

# Create app
mkdir -p rufs.app/Contents/MacOS
go generate ../../version
GOOS=darwin GOARCH=amd64 go build -tags withversion -o rufs.app/Contents/MacOS ../../client

cat <<EOF >rufs.app/Contents/Info.plist
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple Computer//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleGetInfoString</key>
  <string>RUFS</string>
  <key>CFBundleExecutable</key>
  <string>client</string>
  <key>CFBundleIdentifier</key>
  <string>com.github.sgielen.rufs</string>
  <key>CFBundleName</key>
  <string>RUFS</string>
  <key>CFBundleIconFile</key>
  <string>rufs.icns</string>
  <key>CFBundleShortVersionString</key>
  <string>${RUFS_VERSION}</string>
  <key>CFBundleInfoDictionaryVersion</key>
  <string>6.0</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>IFMajorVersion</key>
  <integer>0</integer>
  <key>IFMinorVersion</key>
  <integer>1</integer>
  <!-- avoid having a blurry icon and text -->
  <key>NSHighResolutionCapable</key>
  <string>True</string>
  <!-- avoid showing the app on the Dock -->
  <key>LSUIElement</key>
  <string>1</string>
</dict>
</plist>
EOF

mkdir -p rufs.app/Contents/Resources
cp rufs.icns rufs.app/Contents/Resources
cp com.github.sgielen.rufs.plist rufs.app/Contents/Resources

# Create pkg
mkdir -p tempdir/root/Applications tempdir/packages
cp -R rufs.app tempdir/root/Applications
pkgutil --expand 'macFUSE 4.2.5.pkg' tempdir/macfuse
pkgutil --flatten tempdir/macfuse/Core.pkg tempdir/packages/MacfuseCore.pkg
pkgutil --flatten tempdir/macfuse/PreferencePane.pkg tempdir/packages/MacfusePreferencePane.pkg
pkgbuild \
	--identifier com.github.sgielen.rufs \
	--version "${RUFS_VERSION}" \
	--scripts pkg-scripts \
	--root tempdir/root \
	--component-plist components.plist \
	--install-location / \
	tempdir/packages/rufs-client.pkg
productbuild --distribution Distribution --resources pkg-resources --package-path tempdir/packages rufs-setup.pkg
rm -rf tempdir
