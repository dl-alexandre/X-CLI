class X < Formula
  desc "Terminal-first CLI for X/Twitter with AI-friendly outputs"
  homepage "https://github.com/dl-alexandre/X-CLI"
  version "1.0.0"
  license "MIT"
  head "https://github.com/dl-alexandre/X-CLI.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/x"
    
    # Generate shell completions
    generate_completions_from_executable(bin/"x", "completion", shells: [:bash, :zsh, :fish])
  end

  test do
    # Test version command
    assert_match "x version", shell_output("#{bin}/x version 2>&1", 0)
    
    # Test help command
    assert_match "terminal-first CLI", shell_output("#{bin}/x --help")
    
    # Test doctor command (should work without auth)
    output = shell_output("#{bin}/x doctor 2>&1")
    assert_match "X-CLI Doctor", output
  end
end
