#!/bin/bash
set -e

# Protohost Release Builder
# Builds binaries for multiple platforms

VERSION=${1:-0.1.0}
OUTPUT_DIR="dist"

echo "Building protohost v${VERSION} for multiple platforms..."
echo ""

# Clean and create output directory
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

# Build for different platforms
platforms=(
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
    "linux/arm64"
)

for platform in "${platforms[@]}"; do
    IFS='/' read -r -a parts <<< "$platform"
    GOOS="${parts[0]}"
    GOARCH="${parts[1]}"

    output_name="protohost-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        output_name="${output_name}.exe"
    fi

    echo "Building for ${GOOS}/${GOARCH}..."
    env GOOS="$GOOS" GOARCH="$GOARCH" go build \
        -ldflags="-s -w -X main.version=${VERSION}" \
        -o "${OUTPUT_DIR}/${output_name}" \
        ./cmd/protohost

    # Create tarball
    tar -czf "${OUTPUT_DIR}/${output_name}.tar.gz" -C "$OUTPUT_DIR" "$output_name"
    rm "${OUTPUT_DIR}/${output_name}"

    echo "  ✓ Created ${output_name}.tar.gz"
done

# Create checksums
echo ""
echo "Generating checksums..."
cd "$OUTPUT_DIR"
shasum -a 256 *.tar.gz > checksums.txt
cd ..

echo ""
echo "✅ Build complete! Binaries in ${OUTPUT_DIR}/"
echo ""
ls -lh "${OUTPUT_DIR}"
