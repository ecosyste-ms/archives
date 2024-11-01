class Archive
  attr_accessor :url

  def initialize(url)
    @url = url
  end

  def download_file(dir)
    path = working_directory(dir)
    downloaded_file = File.open(path, "wb")

    request = Typhoeus::Request.new(url, followlocation: true)
    request.on_headers do |response|
      if response.headers && response.headers['Content-Length'] && response.headers['Content-Length'].to_i > 100 * 1024 * 1024
        puts "File is larger than 100MB, aborting download."
        return false
      end
      return nil unless [200,301,302].include? response.code
    end
    request.on_body { |chunk| downloaded_file.write(chunk) }
    request.on_complete { downloaded_file.close }
    request.run
  end

  require 'timeout'

  def extract(dir)
    path = working_directory(dir)
    destination = nil

    if File.size(path) > 100 * 1024 * 1024
      puts "File is larger than 100MB, skipping extraction."
      return nil
    end

    begin
      Timeout::timeout(30) do
        case mime_type(path)
        when "application/zip", "application/java-archive"
          destination = File.join(dir, 'zip')
          `mkdir #{destination} && bsdtar --strip-components=1 -xvf #{path} -C #{destination} > /dev/null 2>&1 `
        when "application/gzip"
          destination = File.join(dir, 'tar')
          `mkdir #{destination} && tar xzf #{path} -C #{destination} --strip-components 1`
        when "application/x-tar"
          if extension == '.gem' # rubygems
            destination = File.join(dir, 'tar')
            data_destination = File.join(dir, 'data')
            data_path = File.join(destination, 'data.tar.gz')
            `mkdir #{destination} && tar xf #{path} -C #{destination} && mkdir #{data_destination} && tar xzf #{data_path} -C #{data_destination}`
            destination = data_destination
          elsif domain == 'repo.hex.pm' # elixir
            destination = File.join(dir, 'tar')
            data_destination = File.join(dir, 'data')
            data_path = File.join(destination, 'contents.tar.gz')
            `mkdir #{destination} && tar xf #{path} -C #{destination} && mkdir #{data_destination} && tar xzf #{data_path} -C #{data_destination}`
            destination = data_destination
          else
            destination = File.join(dir, 'tar')
            `mkdir #{destination} && tar xf #{path} -C #{destination}`
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
        return {
          name: file_path,
          directory: false,
          contents: File.read(full_path)
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

      raw = File.read(File.join(base_path, readme_file))
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

      changelog_files = all_files.select{|path| path.match(/^CHANGE|^HISTORY|^NEWS/i) }.sort{|path| supported_readme_format?(path) ? 0 : 1 }

      changelog_files = changelog_files.sort_by(&:length)
      return nil if changelog_files.empty?

      changelog_file = changelog_files.first

      raw = File.read(File.join(base_path, changelog_file))
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
      `cd #{base_path} && repopack .`
      repopack_output = File.read(File.join(base_path, 'repopack-output.txt'))

      return {
        output: repopack_output
      }
    end
  end
end