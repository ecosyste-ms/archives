# CommonMarker 2.x compatibility
# In CommonMarker 2.x, the constant name changed from CommonMarker to Commonmarker
# and the API changed significantly. This provides backward compatibility.
require 'commonmarker'

if defined?(Commonmarker) && !defined?(CommonMarker)
  # Create the CommonMarker constant
  CommonMarker = Commonmarker
  
  # Add backward compatibility methods
  module CommonMarker
    def self.render_html(text, options = [], extensions = [])
      # Convert old-style options array to new options hash
      opts = {}
      
      # Handle common options from CommonMarker 0.x
      if options.is_a?(Array)
        options.each do |opt|
          case opt
          when :GITHUB_PRE_LANG
            opts[:github_pre_lang] = true
          when :HARDBREAKS
            opts[:hardbreaks] = true
          when :UNSAFE
            opts[:unsafe] = true
          when :SOURCEPOS
            opts[:sourcepos] = true
          when :VALIDATE_UTF8
            opts[:validate_utf8] = true
          when :SMART
            opts[:smart] = true
          when :LIBERAL_HTML_TAG
            opts[:liberal_html_tag] = true
          when :FOOTNOTES
            opts[:footnotes] = true
          when :STRIKETHROUGH_DOUBLE_TILDE
            opts[:strikethrough_double_tilde] = true
          end
        end
      elsif options.is_a?(Hash)
        opts = options
      end
      
      # Convert extensions array to plugins
      plugins = {}
      if extensions.is_a?(Array)
        extensions.each do |ext|
          case ext
          when :tagfilter
            plugins[:tagfilter] = true
          when :autolink
            plugins[:autolink] = true
          when :table
            plugins[:table] = true
          when :strikethrough
            plugins[:strikethrough] = true
          when :tasklist
            plugins[:tasklist] = true
          end
        end
      end
      
      # CommonMarker 2.x uses to_html method with different parameters
      Commonmarker.to_html(text, options: opts, plugins: plugins)
    end
    
    def self.render_doc(text, options = [], extensions = [])
      # Similar conversion for render_doc
      opts = {}
      plugins = {}
      
      if options.is_a?(Array)
        options.each do |opt|
          case opt
          when :GITHUB_PRE_LANG
            opts[:github_pre_lang] = true
          when :HARDBREAKS
            opts[:hardbreaks] = true
          when :UNSAFE
            opts[:unsafe] = true
          when :SOURCEPOS
            opts[:sourcepos] = true
          when :VALIDATE_UTF8
            opts[:validate_utf8] = true
          when :SMART
            opts[:smart] = true
          end
        end
      elsif options.is_a?(Hash)
        opts = options
      end
      
      if extensions.is_a?(Array)
        extensions.each do |ext|
          case ext
          when :tagfilter
            plugins[:tagfilter] = true
          when :autolink
            plugins[:autolink] = true
          when :table
            plugins[:table] = true
          when :strikethrough
            plugins[:strikethrough] = true
          when :tasklist
            plugins[:tasklist] = true
          end
        end
      end
      
      Commonmarker.parse(text, options: opts, plugins: plugins)
    end
  end
end