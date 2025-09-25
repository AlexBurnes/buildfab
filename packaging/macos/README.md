# macOS Homebrew Tap for buildfab CLI

This directory contains the Homebrew formula for the buildfab CLI utility.

## Installation

To install the buildfab CLI via Homebrew:

```bash
# Add the tap
brew tap AlexBurnes/homebrew-tap https://github.com/AlexBurnes/homebrew-tap

# Install the formula
brew install buildfab
```

## Updating

To update to the latest version:

```bash
brew update
brew upgrade buildfab
```

## Uninstalling

To remove the buildfab CLI:

```bash
brew uninstall buildfab
```

## Formula Details

- **Formula Name**: `buildfab`
- **Description**: Go-based automation runner with DAG executor
- **License**: MIT
- **Homepage**: https://github.com/AlexBurnes/buildfab
- **Architectures**: amd64, arm64 (Apple Silicon)

## Manual Installation

If you prefer not to use Homebrew, you can download the binary directly:

```bash
# For Intel Macs
curl -L https://github.com/AlexBurnes/buildfab/releases/latest/download/buildfab_darwin_amd64.tar.gz | tar -xz
sudo mv buildfab /usr/local/bin/

# For Apple Silicon Macs
curl -L https://github.com/AlexBurnes/buildfab/releases/latest/download/buildfab_darwin_arm64.tar.gz | tar -xz
sudo mv buildfab /usr/local/bin/
```

## Using Installer Scripts

You can also use the self-extracting installer scripts:

```bash
# For Intel Macs
wget -O - https://github.com/AlexBurnes/buildfab/releases/latest/download/buildfab-darwin-amd64-install.sh | sh

# For Apple Silicon Macs
wget -O - https://github.com/AlexBurnes/buildfab/releases/latest/download/buildfab-darwin-arm64-install.sh | sh
```

## Verification

After installation, verify the installation:

```bash
buildfab --version
buildfab --help
```

## Support

For issues and support, please visit the [main project repository](https://github.com/AlexBurnes/buildfab).