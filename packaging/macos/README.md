# macOS Homebrew Tap for Version CLI

This directory contains the Homebrew formula for the Version CLI utility.

## Installation

To install the Version CLI via Homebrew:

```bash
# Add the tap
brew tap AlexBurnes/homebrew-tap https://github.com/AlexBurnes/homebrew-tap

# Install the formula
brew install version
```

## Updating

To update to the latest version:

```bash
brew update
brew upgrade version
```

## Uninstalling

To remove the Version CLI:

```bash
brew uninstall version
```

## Formula Details

- **Formula Name**: `version`
- **Description**: Cross-platform semantic version parsing, validation, and ordering CLI utility
- **License**: Apache-2.0
- **Homepage**: https://github.com/AlexBurnes/version-go
- **Architectures**: amd64, arm64 (Apple Silicon)

## Manual Installation

If you prefer not to use Homebrew, you can download the binary directly:

```bash
# For Intel Macs
curl -L https://github.com/AlexBurnes/version-go/releases/latest/download/version-macos-amd64.tar.gz | tar -xz
sudo mv version /usr/local/bin/

# For Apple Silicon Macs
curl -L https://github.com/AlexBurnes/version-go/releases/latest/download/version-macos-arm64.tar.gz | tar -xz
sudo mv version /usr/local/bin/
```

## Using Installer Scripts

You can also use the self-extracting installer scripts:

```bash
# For Intel Macs
wget -O - https://github.com/AlexBurnes/version-go/releases/latest/download/version-macos-amd64-install.sh | sh

# For Apple Silicon Macs
wget -O - https://github.com/AlexBurnes/version-go/releases/latest/download/version-macos-arm64-install.sh | sh
```

## Verification

After installation, verify the installation:

```bash
version --version
version --help
```

## Support

For issues and support, please visit the [main project repository](https://github.com/AlexBurnes/version-go).