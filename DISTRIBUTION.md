# Distributing Protohost

This document explains how to package and distribute protohost to other developers.

## Quick Distribution Methods

### Option 1: Pre-built Binaries (Recommended)

**Best for:** Quick distribution within your company

1. **Build binaries for all platforms:**
   ```bash
   ./build-release.sh 0.1.0
   ```
   This creates binaries in `dist/`:
   - `protohost-darwin-amd64.tar.gz` (macOS Intel)
   - `protohost-darwin-arm64.tar.gz` (macOS Apple Silicon)
   - `protohost-linux-amd64.tar.gz` (Linux x86_64)
   - `protohost-linux-arm64.tar.gz` (Linux ARM64)

2. **Upload to internal file server or shared drive:**
   - Upload the `dist/` folder to your company's file server
   - Share the download link with your team

3. **Install on developer machines:**
   ```bash
   # Download appropriate binary
   curl -O https://your-server.com/protohost-darwin-arm64.tar.gz

   # Extract and install
   tar -xzf protohost-darwin-arm64.tar.gz
   sudo mv protohost-darwin-arm64 /usr/local/bin/protohost
   chmod +x /usr/local/bin/protohost

   # Verify
   protohost --version
   ```

### Option 2: GitHub/GitLab Releases

**Best for:** Version-controlled distribution

1. **Create a release on GitHub/GitLab:**
   ```bash
   # Build binaries
   ./build-release.sh 0.1.0

   # Create git tag
   git tag v0.1.0
   git push origin v0.1.0

   # Upload binaries to release
   # (Use GitHub UI or gh CLI)
   ```

2. **Developers install via curl:**
   ```bash
   curl -sSL https://github.com/your-org/protohost/releases/download/v0.1.0/protohost-$(uname -s)-$(uname -m).tar.gz | tar xz
   sudo mv protohost-* /usr/local/bin/protohost
   ```

### Option 3: Build from Source

**Best for:** Open development environments

Share the repository:
```bash
git clone git@github.com:your-org/protohost.git
cd protohost
make install
```

### Option 4: Docker Image

**Best for:** Containerized workflows

Create a `Dockerfile`:
```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o protohost cmd/protohost/main.go

FROM alpine:latest
RUN apk add --no-cache git docker-cli openssh-client
COPY --from=builder /app/protohost /usr/local/bin/protohost
ENTRYPOINT ["protohost"]
```

Build and push:
```bash
docker build -t your-registry.com/protohost:0.1.0 .
docker push your-registry.com/protohost:0.1.0
```

Developers use:
```bash
docker run --rm -it \
  -v ~/.ssh:/root/.ssh:ro \
  -v $(pwd):/workspace \
  -w /workspace \
  your-registry.com/protohost:0.1.0 deploy
```

## Quick Start Guide for Developers

Share this with your team:

---

## Installing Protohost

**Method 1: Pre-built binary (fastest)**
```bash
# Download for your platform
curl -O https://your-server.com/protohost-darwin-arm64.tar.gz

# Extract and install
tar -xzf protohost-darwin-arm64.tar.gz
sudo mv protohost-darwin-arm64 /usr/local/bin/protohost
chmod +x /usr/local/bin/protohost

# Verify
protohost --version
```

**Method 2: Build from source**
```bash
git clone git@github.com:your-org/protohost.git
cd protohost
make install
```

**Method 3: Install script**
```bash
curl -sSL https://your-server.com/install.sh | bash
```

## Using Protohost

**Initialize in your project:**
```bash
cd your-project
protohost init
```

**Edit `.protohost.config`:**
```bash
PROJECT_PREFIX="myapp"
REPO_URL="git@github.com:your-org/your-repo.git"
REMOTE_HOST="protohost.example.com"
REMOTE_USER="your-username"
# ... etc
```

**Deploy locally:**
```bash
protohost deploy
```

**Deploy to staging/production:**
```bash
protohost deploy --remote
```

**View logs:**
```bash
protohost logs -f
```

**List deployments:**
```bash
protohost list
protohost list --remote
```

---

## Internal Distribution Setup

### 1. Create Internal Package Repository

**Using Artifactory/Nexus:**
```bash
# Upload binaries
./build-release.sh 0.1.0
curl -u user:pass -T dist/protohost-darwin-arm64.tar.gz \
  https://artifactory.example.com/protohost/0.1.0/
```

**Using AWS S3:**
```bash
# Upload to S3
aws s3 sync dist/ s3://your-company-binaries/protohost/0.1.0/
```

**Using Internal Web Server:**
```bash
# Copy to web server
scp -r dist/ webserver:/var/www/downloads/protohost/0.1.0/
```

### 2. Create Install Script

Customize `install.sh` with your internal URLs:
```bash
# Update these lines:
GITHUB_REPO="your-org/protohost"
DOWNLOAD_URL="https://internal-server.com/protohost/${VERSION}/${BINARY_NAME}.tar.gz"
```

### 3. Document Installation

Create internal wiki page or README:
```markdown
# Protohost Installation

Quick install:
\`\`\`bash
curl -sSL https://internal-server.com/protohost/install.sh | bash
\`\`\`

Or download manually from:
https://internal-server.com/protohost/latest/
```

## Version Management

### Semantic Versioning
- `0.1.0` - Initial release
- `0.2.0` - New features
- `0.2.1` - Bug fixes
- `1.0.0` - Stable release

### Update Process
```bash
# Build new version
./build-release.sh 0.2.0

# Tag release
git tag v0.2.0
git push origin v0.2.0

# Upload binaries
# (to your distribution method)

# Update install script to point to new version
```

## Troubleshooting

### Binary won't run on macOS
```bash
# Remove quarantine attribute
xattr -d com.apple.quarantine /usr/local/bin/protohost
```

### Permission denied
```bash
# Make executable
chmod +x /usr/local/bin/protohost
```

### Command not found
```bash
# Add to PATH
echo 'export PATH="/usr/local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

## Support

For issues or questions:
- Internal Slack: #protohost-support
- Create issue: https://github.com/your-org/protohost/issues
- Contact: devops@example.com
