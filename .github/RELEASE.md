# Release Process

This document describes how to create a new release of protohost.

## Automated Release via GitHub Actions

The GitHub Actions workflow automatically builds and releases protohost when you push a version tag.

### Creating a Release

1. **Update version and commit changes:**
   ```bash
   # Update CHANGELOG.md with release notes
   vim CHANGELOG.md

   git add .
   git commit -m "Prepare v0.2.0 release"
   git push origin main
   ```

2. **Create and push a version tag:**
   ```bash
   git tag v0.2.0
   git push origin v0.2.0
   ```

3. **GitHub Actions will automatically:**
   - Build binaries for all platforms (macOS, Linux, ARM64, AMD64)
   - Run tests
   - Create a GitHub release
   - Upload binaries and checksums
   - Generate release notes from commits

4. **Verify the release:**
   - Go to https://github.com/your-org/protohost/releases
   - Check that binaries are uploaded
   - Verify checksums

### Manual Build (if needed)

If you need to build locally:

```bash
# Build all platforms
./build-release.sh 0.2.0

# Binaries will be in dist/
ls -lh dist/
```

## Version Numbers

We use [Semantic Versioning](https://semver.org/):

- **MAJOR.MINOR.PATCH** (e.g., 1.2.3)
- **MAJOR**: Breaking changes
- **MINOR**: New features (backwards compatible)
- **PATCH**: Bug fixes (backwards compatible)

Examples:
- `v0.1.0` - Initial release
- `v0.2.0` - New features added
- `v0.2.1` - Bug fixes
- `v1.0.0` - First stable release

## Testing a Release

Before creating a release, test the build:

```bash
# Build locally
make build

# Test basic functionality
./protohost --version
./protohost --help

# Test in a project
cd /tmp/test-project
/path/to/protohost init
/path/to/protohost deploy
```

## Release Checklist

- [ ] Update CHANGELOG.md
- [ ] Commit all changes
- [ ] Create and push version tag
- [ ] Wait for GitHub Actions to complete
- [ ] Verify release on GitHub
- [ ] Test installation from release
- [ ] Update documentation if needed
- [ ] Announce release (Slack, email, etc.)

## Installation from Release

Once released, users can install via:

```bash
# Download and install
VERSION="0.2.0"
OS="darwin"  # or linux
ARCH="arm64"  # or amd64

curl -LO "https://github.com/your-org/protohost/releases/download/v${VERSION}/protohost-${OS}-${ARCH}.tar.gz"
tar -xzf "protohost-${OS}-${ARCH}.tar.gz"
sudo mv "protohost-${OS}-${ARCH}" /usr/local/bin/protohost
chmod +x /usr/local/bin/protohost

# Verify
protohost --version
```

Or use the install script:
```bash
curl -sSL https://raw.githubusercontent.com/your-org/protohost/main/install.sh | bash
```

## Rollback

If a release has issues:

1. Delete the tag and release from GitHub
2. Fix the issues
3. Create a new patch release

```bash
# Delete tag locally and remotely
git tag -d v0.2.0
git push origin :refs/tags/v0.2.0

# Fix issues, then create new tag
git tag v0.2.1
git push origin v0.2.1
```

## Pre-releases

For testing before official release:

```bash
# Create a pre-release tag
git tag v0.2.0-rc1
git push origin v0.2.0-rc1
```

Mark the GitHub release as "pre-release" for testing.
