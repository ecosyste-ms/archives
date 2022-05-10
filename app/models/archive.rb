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
      return nil if response.code != 200
    end
    request.on_body { |chunk| downloaded_file.write(chunk) }
    request.on_complete { downloaded_file.close }
    request.run
  end

  def extract(dir)
    path = working_directory(dir)

    case mime_type(path)
    when "application/zip"
      destination = File.join(dir, 'zip')
      `unzip -qj #{path} -d #{destination}`
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
end