class Spacedock < Formula
  desc "Workflow launcher and first-officer dispatch for agentic dev"
  homepage "https://github.com/spacedock-dev/spacedock"
  version "0.19.0"
  license "Apache-2.0"

  on_macos do
    on_arm do
      url "https://github.com/spacedock-dev/spacedock/releases/download/v#{version}/spacedock_#{version}_darwin_arm64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
    on_intel do
      url "https://github.com/spacedock-dev/spacedock/releases/download/v#{version}/spacedock_#{version}_darwin_amd64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
  end

  def install
    bin.install "spacedock"
  end

  def caveats
    <<~EOS
      spacedock launches host agent tooling (Claude Code / Codex) inside a
      sandbox called "safehouse". safehouse is a runtime dependency and is NOT
      installed by brew; install it through the host tooling before first launch.
      See the safehouse install docs:
        https://github.com/spacedock-dev/safehouse#install
    EOS
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/spacedock --version")
  end
end
