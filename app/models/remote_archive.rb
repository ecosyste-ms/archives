require 'timeout'
require 'fileutils'
require 'zip'
require 'zlib'
require 'minitar'

class RemoteArchive
  attr_accessor :url

  def initialize(url)
    @url = url
    uri = URI.parse(url)
    unless ["http", "https"].include?(uri.scheme)
      raise ArgumentError, "Only HTTP/HTTPS URLs are allowed"
    end
  end

  def download_file(dir)
    path = working_directory(dir)

    response = Typhoeus.get(url, followlocation: true)

    unless [200, 301, 302].include?(response.code)
      Rails.logger.info("Download failed: HTTP #{response.code}")
      return nil
    end

    if response.body.bytesize > 100 * 1024 * 1024
      Rails.logger.info("File is larger than 100MB, skipping extraction.")
      return false
    end

    File.binwrite(path, response.body)
  end

  def extract(dir)
    path = working_directory(dir)
    destination = nil

    if File.size(path) > 100 * 1024 * 1024
      Rails.logger.info("File is larger than 100MB, skipping extraction.")
      return nil
    end

    begin
      Timeout::timeout(30) do
        case mime_type(path)
        when "application/zip", "application/java-archive"
          destination = File.join(dir, 'zip')
          FileUtils.mkdir_p(destination)
          Zip::File.open(path) do |zip_file|
            file_count = 0
            
            # Check if we should strip top-level directory by looking at entry structure
            all_entries = zip_file.entries.reject { |entry| entry.respond_to?(:symlink?) && entry.symlink? }
            non_root_entries = all_entries.reject { |entry| entry.name.split(File::SEPARATOR).length == 1 && entry.directory? }
            should_strip_top = non_root_entries.all? { |entry| entry.name.split(File::SEPARATOR).length > 1 } && 
                             all_entries.map { |entry| entry.name.split(File::SEPARATOR).first }.uniq.length == 1 &&
                             non_root_entries.any?
            
            zip_file.each do |entry|
              next if entry.respond_to?(:symlink?) && entry.symlink?
              components = entry.name.split(File::SEPARATOR)
              next if components.empty?
              
              # Only strip top-level folder if all entries share the same top-level folder
              stripped_path = should_strip_top ? File.join(components.drop(1)) : entry.name
              next if stripped_path.empty?
              
              entry_path = File.join(destination, stripped_path)
              entry_path = File.expand_path(entry_path)
              raise "Blocked extraction outside target dir" unless entry_path.start_with?(File.expand_path(destination))
              raise "Too many files in archive" if (file_count += 1) > 10_000
              
              if entry.directory?
                FileUtils.mkdir_p(entry_path)
              else
                FileUtils.mkdir_p(File.dirname(entry_path))
                begin
                  # Extract manually to avoid issues with zip_file.extract
                  File.open(entry_path, 'wb') do |file|
                    file.write(entry.get_input_stream.read)
                  end
                rescue => e
                  Rails.logger.warn("Failed to extract #{entry.name}: #{e.message}")
                  next
                end
              end
            end
          end
        when "application/gzip"
          destination = File.join(dir, 'tar')
          FileUtils.mkdir_p(destination)
          Zlib::GzipReader.open(path) do |gz|
            Minitar::Reader.open(gz) do |reader|
              extract_tar(reader, destination)
            end
          end
        when "application/x-tar"
          destination = File.join(dir, 'tar')
          FileUtils.mkdir_p(destination)
          File.open(path, "rb") do |file|
            Minitar::Reader.open(file) do |reader|
              extract_tar(reader, destination)
            end
          end
          data_destination = File.join(dir, 'data')
          data_path = File.join(destination, 'data.tar.gz')
          FileUtils.mkdir_p(data_destination)
          Zlib::GzipReader.open(data_path) do |gz|
            Minitar::Reader.open(gz) do |reader|
              extract_tar(reader, data_destination)
            end
          end
        else
          # not supported
          destination = nil
        end
      end
    rescue Timeout::Error
      puts "The operation timed out after 30 seconds"
      destination = nil
    end

    return destination
  end

  public

  def mime_type(path)
    IO.popen(
      ["file", "--brief", "--mime-type", path],
      in: :close, err: :close
    ) { |io| io.read.chomp }
  end

  def list_files
    Dir.mktmpdir do |dir|
      download_file(dir)
      path = extract(dir)
      return [] if path.nil?
      return Dir.glob("**/*", File::FNM_DOTMATCH, base: path).tap{|a| a.delete(".")}
    end    
  end

  def working_directory(dir)
    File.join(dir, basename)
  end

  def basename
    File.basename(url)
  end

  def extension
    File.extname(basename)
  end

  def domain
    URI.parse(url).host.downcase
  end

  def contents(file_path)
    Dir.mktmpdir do |dir|
      download_file(dir)
      base_path = extract(dir)
      full_path = File.join(base_path, file_path)
      return nil if base_path.nil?
      begin
        raw_contents = File.read(full_path)
        return {
          name: file_path,
          directory: false,
          contents: raw_contents.force_encoding('UTF-8').scrub('�')
        }
      rescue Errno::EISDIR
        return {
          name: file_path,
          directory: true,
          contents: Dir.glob("**/*", File::FNM_DOTMATCH, base: full_path).tap{|a| a.delete(".")}
        }        
      rescue Errno::ENOENT
        return nil
      end
    end   
  end

  def supported_readme_format?(path)
    [
      /md/i,
      /mdown/i,
      /mkdn/i,
      /mdn/i,
      /mdtext/i,
      /markdown/i,
      /textile/i,
      /org/i,
      /creole/i,
      /mediawiki/i,
      /wiki/i,
      /adoc|asc(iidoc)?/i,
      /re?st(\.txt)?/i,
      /pod/i,
      /rdoc/i
    ].any? do |regexp|
      path =~ /\.(#{regexp})\z/
    end
  end

  def readme
    Dir.mktmpdir do |dir|
      download_file(dir)
      base_path = extract(dir)

      return nil if base_path.nil?
      all_files = Dir.glob("**/*", File::FNM_DOTMATCH, base: base_path).tap{|a| a.delete(".")}

      readme_files = all_files.select{|path| path.match(/^readme/i) }.sort{|path| supported_readme_format?(path) ? 0 : 1 }
      readme_files = readme_files.sort_by(&:length)
      readme_file = readme_files.first

      return nil if readme_file.nil?

      begin
        raw = File.read(File.join(base_path, readme_file))
      rescue Errno::EISDIR
        Rails.logger.info("Skipping readme directory: #{readme_file}")
        return nil
      end
      html = GitHub::Markup.render(readme_file, raw.force_encoding("UTF-8"))
      language = GitHub::Markup.language(readme_file, raw.force_encoding("UTF-8")).try(:name)

      return {
        name: readme_file,
        raw: raw,
        html: html,
        plain: Nokogiri::HTML(html).try(:text),
        extension: File.extname(readme_file),
        language: language,
        other_readme_files: readme_files - [readme_file]
      }
    end
  end

  def changelog
    Dir.mktmpdir do |dir|
      download_file(dir)
      base_path = extract(dir)

      return nil if base_path.nil?
      all_files = Dir.glob("**/*", File::FNM_DOTMATCH, base: base_path).tap{|a| a.delete(".")}

      changelog_files = all_files.select do |path|
        full_path = File.join(base_path, path)
        path.match(/^CHANGE|^HISTORY|^NEWS/i) && File.file?(full_path)
      end
      changelog_files = changelog_files.sort{|path| supported_readme_format?(path) ? 0 : 1 }
      changelog_files = changelog_files.sort_by(&:length)
      return nil if changelog_files.empty?

      changelog_file = changelog_files.first

      begin
        raw = File.read(File.join(base_path, changelog_file))
      rescue Errno::EISDIR
        Rails.logger.info("Skipping changelog directory: #{changelog_file}")
        return nil
      end
      html = GitHub::Markup.render(changelog_file, raw.force_encoding("UTF-8"))
      language = GitHub::Markup.language(changelog_file, raw.force_encoding("UTF-8")).try(:name)

      return {
        name: changelog_file,
        raw: raw,
        html: html,
        plain: Nokogiri::HTML(html).try(:text),
        parsed: Vandamme::Parser.new(changelog: raw).parse,
        extension: File.extname(changelog_file),
        language: language,
        other_readme_files: changelog_files - [changelog_file]
      }
    end
  end
  
  def repopack
    Dir.mktmpdir do |dir|
      download_file(dir)
      base_path = extract(dir)

      return nil if base_path.nil?
      cmd = "cd #{Shellwords.escape(base_path)} && repomix . --output repomix-output.txt --verbose"
      system(cmd)
      repopack_output = File.read(File.join(base_path, 'repomix-output.txt'))

      return {
        output: repopack_output
      }
    end
  end

  private

  def extract_tar(reader, destination)
    file_count = 0
    reader.each_entry do |entry|
      next if entry.respond_to?(:symlink?) && entry.symlink?
      components = entry.name.split(File::SEPARATOR)
      next if components.empty?
      stripped_path = File.join(components.drop(1))
      next if stripped_path.empty?
      dest_path = File.join(destination, stripped_path)
      dest_path = File.expand_path(dest_path)
      raise "Blocked extraction outside target dir" unless dest_path.start_with?(File.expand_path(destination))
      raise "Too many files in archive" if (file_count += 1) > 10_000
      if entry.directory?
        FileUtils.mkdir_p(dest_path)
      else
        FileUtils.mkdir_p(File.dirname(dest_path))
        File.open(dest_path, "wb") do |f|
          while (chunk = entry.read(4096))
            f.write(chunk)
          end
        end
      end
    end
  end
end