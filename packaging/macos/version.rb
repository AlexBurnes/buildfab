class Buildfab < Formula
  desc "Go-based automation runner with DAG executor"
  homepage "https://github.com/AlexBurnes/buildfab"
  url "https://github.com/AlexBurnes/buildfab/releases/download/v0.8.8/buildfab_macos_amd64.tar.gz"
  sha256 "PLACEHOLDER_SHA256"
  license "MIT"
  
  if Hardware::CPU.arm?
    url "https://github.com/AlexBurnes/buildfab/releases/download/v0.8.8/buildfab_macos_arm64.tar.gz"
    sha256 "PLACEHOLDER_SHA256_ARM64"
  end

  def install
    bin.install "buildfab"
    man1.install "buildfab.1" if File.exist?("buildfab.1")
  end

  test do
    assert_match "buildfab", shell_output("#{bin}/buildfab --version").strip
    assert_match "buildfab", shell_output("#{bin}/buildfab --help").strip
    assert_match "list-actions", shell_output("#{bin}/buildfab list-actions").strip
  end
end