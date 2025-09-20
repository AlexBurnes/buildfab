class Version < Formula
  desc "Cross-platform semantic version parsing, validation, and ordering CLI utility"
  homepage "https://github.com/AlexBurnes/version-go"
  url "https://github.com/AlexBurnes/version-go/releases/download/v0.5.9/version-0.5.9-darwin-amd64.tar.gz"
  sha256 "PLACEHOLDER_SHA256"
  license "Apache-2.0"
  
  if Hardware::CPU.arm?
    url "https://github.com/AlexBurnes/version-go/releases/download/v0.5.9/version-0.5.9-darwin-arm64.tar.gz"
    sha256 "PLACEHOLDER_SHA256_ARM64"
  end

  def install
    bin.install "version"
    man1.install "version.1" if File.exist?("version.1")
  end

  test do
    assert_equal "0.5.4", shell_output("#{bin}/version --version").strip
    assert_equal "release", shell_output("#{bin}/version type 1.2.3").strip
    assert_equal "prerelease", shell_output("#{bin}/version type 1.2.3~alpha.1").strip
    assert_equal "postrelease", shell_output("#{bin}/version type 1.2.3.fix.1").strip
    assert_equal "intermediate", shell_output("#{bin}/version type 1.2.3_feature_1").strip
  end
end