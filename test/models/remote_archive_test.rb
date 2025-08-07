require "test_helper"
require "remote_archive"

class ArchiveTest < ActiveSupport::TestCase
  test "initializes with valid http url" do
    archive = RemoteArchive.new("http://example.com/file.zip")
    assert_equal "http://example.com/file.zip", archive.url
  end

  test "raises with invalid scheme" do
    assert_raises(ArgumentError) do
      RemoteArchive.new("ftp://example.com/file.zip")
    end
  end

  test "basename returns correct filename" do
    archive = RemoteArchive.new("https://example.com/files/foo.tar.gz")
    assert_equal "foo.tar.gz", archive.basename
  end

  test "extension returns correct file extension" do
    archive = RemoteArchive.new("https://example.com/files/foo.tar.gz")
    assert_equal ".gz", archive.extension
  end

  test "domain returns correct host" do
    archive = RemoteArchive.new("https://repo.hex.pm/tarball/package-1.0.0")
    assert_equal "repo.hex.pm", archive.domain
  end

  test "working_directory joins dir and basename" do
    archive = RemoteArchive.new("https://example.com/thing.zip")
    path = archive.working_directory("/tmp")
    assert_equal "/tmp/thing.zip", path
  end

  test "supported_readme_format? matches .md" do
    archive = RemoteArchive.new("http://example.com/foo.zip")
    assert archive.supported_readme_format?("README.md")
    refute archive.supported_readme_format?("README.exe")
  end

  # Integration-y tests (e.g. list_files, extract, readme, changelog, etc)
  # These would normally use fixtures or VCR to avoid actual downloads.
  # Here’s a placeholder test to show intent:

  test "extracts tar.gz file correctly" do
    fixture_path = Rails.root.join("test/fixtures/files/base62-2.0.1.tgz")
    archive = RemoteArchive.new("http://example.com/base62.tgz")

    Dir.mktmpdir do |dir|
      dest = archive.working_directory(dir)
      FileUtils.cp(fixture_path, dest)

      archive.stubs(:download_file).returns(dest)

      destination = archive.extract(dir)
      assert destination.present?, "Expected extract to return a destination"
      files = Dir.glob("**/*", base: destination)
      assert files.any?, "Expected some files to be extracted"
    end
  end

  test "returns nil for files larger than 100MB" do
    archive = RemoteArchive.new("http://example.com/base62.tgz")

    Dir.mktmpdir do |dir|
      dest = archive.working_directory(dir)
      File.open(dest, "wb") { |f| f.write("0" * 101 * 1024 * 1024) }

      archive.stubs(:download_file).returns(dest)

      assert_nil archive.extract(dir), "Expected nil for files larger than 100MB"
    end
  end

  test "handles unsupported mime type" do
    fixture_path = Rails.root.join("test/fixtures/files/base62-2.0.1.tgz")
    archive = RemoteArchive.new("http://example.com/base62.tgz")

    Dir.mktmpdir do |dir|
      dest = archive.working_directory(dir)
      FileUtils.cp(fixture_path, dest)

      archive.stubs(:download_file).returns(dest)
      archive.stubs(:mime_type).returns("application/unknown")

      assert_nil archive.extract(dir), "Expected nil for unsupported mime types"
    end
  end

  test "blocks archive paths escaping destination" do
    archive = RemoteArchive.new("http://example.com/evil.tar.gz")

    Dir.mktmpdir do |dir|
      dest = archive.working_directory(dir)
      Zlib::GzipWriter.open(dest) do |gz|
        Minitar::Writer.open(gz) do |tar|
          tar.add_file_simple("../../evil.txt", :mode => 0644, :size => 5) do |io|
            io.write("oops!")
          end
        end
      end

      archive.stubs(:download_file).returns(dest)
      assert_raises(RuntimeError, "Blocked extraction outside target dir") do
        archive.extract(dir)
      end
    end
  end

  test "uses configured user agent in HTTP requests" do
    archive = RemoteArchive.new("https://example.com/test.zip")
    
    # Mock the Typhoeus request to capture the user agent
    request = mock()
    request.expects(:on_headers).returns(request)
    request.expects(:on_body).returns(request)
    request.expects(:on_complete).returns(request)
    request.expects(:run)
    
    Typhoeus::Request.expects(:new).with(
      "https://example.com/test.zip",
      followlocation: true
    ).returns(request)
    
    # Verify the global user agent is set
    assert_equal "archives.ecosyste.ms", Typhoeus::Config.user_agent
    
    Dir.mktmpdir do |dir|
      archive.download_file(dir)
    end
  end

  test "handles file extraction errors gracefully" do
    archive = RemoteArchive.new("http://example.com/test.zip")

    Dir.mktmpdir do |dir|
      dest = archive.working_directory(dir)
      
      # Create an empty file to simulate a zip file
      File.write(dest, "fake zip content")

      # Mock a zip entry that will cause an ENOENT error during extraction
      zip_file = mock()
      entry = mock()
      entry.stubs(:respond_to?).with(:symlink?).returns(false)
      entry.stubs(:name).returns("problematic_file.txt")
      entry.stubs(:directory?).returns(false)
      
      zip_file.stubs(:entries).returns([entry])
      zip_file.stubs(:each).yields(entry)
      zip_file.expects(:extract).with(entry, anything).raises(Errno::ENOENT.new("No such file"))

      # Mock the mime_type and file opening
      archive.stubs(:mime_type).returns("application/zip")
      Zip::File.expects(:open).with(dest).yields(zip_file)

      # Mock Rails.logger to capture the warning
      Rails.logger.expects(:warn).with(regexp_matches(/Failed to extract problematic_file.txt/))

      archive.stubs(:download_file).returns(dest)
      
      # Should not raise an error and should return a path
      result = archive.extract(dir)
      assert_not_nil result, "Extract should still return a destination even with file errors"
    end
  end
end