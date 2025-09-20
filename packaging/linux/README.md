# Linux Installation

## Method 1: Using Simple Installer Scripts

Download and run the installer script directly:

```bash
# Download and install to /usr/local/bin (requires sudo)
# For x86_64/amd64 systems:
wget -O - https://github.com/AlexBurnes/version-go/releases/latest/download/version-linux-amd64-install.sh | sh

# For ARM64 systems:
wget -O - https://github.com/AlexBurnes/version-go/releases/latest/download/version-linux-arm64-install.sh | sh

# Download and install to custom directory
# For x86_64/amd64 systems:
INSTALL_DIR=/opt/version wget -O - https://github.com/AlexBurnes/version-go/releases/latest/download/version-linux-amd64-install.sh | sh

# For ARM64 systems:
INSTALL_DIR=/opt/version wget -O - https://github.com/AlexBurnes/version-go/releases/latest/download/version-linux-arm64-install.sh | sh

# Or download first, then install
# For x86_64/amd64 systems:
wget https://github.com/AlexBurnes/version-go/releases/latest/download/version-linux-amd64-install.sh
chmod +x version-linux-amd64-install.sh
./version-linux-amd64-install.sh /usr/local/bin

# For ARM64 systems:
wget https://github.com/AlexBurnes/version-go/releases/latest/download/version-linux-arm64-install.sh
chmod +x version-linux-arm64-install.sh
./version-linux-arm64-install.sh /usr/local/bin
```

## Method 2: Manual Installation from Archive

Download and extract the archive manually:

```bash
# Download and extract archive
wget https://github.com/AlexBurnes/version-go/releases/download/v1.0.0/version_1.0.0_linux_amd64.tar.gz
tar -xzf version_1.0.0_linux_amd64.tar.gz
cd version_1.0.0_linux_amd64

# Install using the included install.sh
./install.sh /usr/local/bin

# Or install to user directory
./install.sh ~/.local/bin
```

## Windows Installation

Windows users can use Scoop:
```bash
scoop install version
```