require "test_helper"
require "archive"

class ArchiveTest < ActiveSupport::TestCase
  test "initializes with valid http url" do
    archive = Archive.new("http://example.com/file.zip")
    assert_equal "http://example.com/file.zip", archive.url
  end

  test "raises with invalid scheme" do
    assert_raises(ArgumentError) do
      Archive.new("ftp://example.com/file.zip")
    end
  end

  test "basename returns correct filename" do
    archive = Archive.new("https://example.com/files/foo.tar.gz")
    assert_equal "foo.tar.gz", archive.basename
  end

  test "extension returns correct file extension" do
    archive = Archive.new("https://example.com/files/foo.tar.gz")
    assert_equal ".gz", archive.extension
  end

  test "domain returns correct host" do
    archive = Archive.new("https://repo.hex.pm/tarball/package-1.0.0")
    assert_equal "repo.hex.pm", archive.domain
  end

  test "working_directory joins dir and basename" do
    archive = Archive.new("https://example.com/thing.zip")
    path = archive.working_directory("/tmp")
    assert_equal "/tmp/thing.zip", path
  end

  test "supported_readme_format? matches .md" do
    archive = Archive.new("http://example.com/foo.zip")
    assert archive.supported_readme_format?("README.md")
    refute archive.supported_readme_format?("README.exe")
  end

  # Integration-y tests (e.g. list_files, extract, readme, changelog, etc)
  # These would normally use fixtures or VCR to avoid actual downloads.
  # Here’s a placeholder test to show intent:

  test "download_file returns false for large files" do
    archive = Archive.new("http://example.com/large.zip")
  
    stub_request(:get, "http://example.com/large.zip")
      .to_return(
        status: 200,
        headers: { "Content-Length" => "#{101 * 1024 * 1024}" }
      )
  
    Dir.mktmpdir do |dir|
      result = archive.download_file(dir)
      assert_equal false, result
    end
  end
end