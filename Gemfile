source "https://rubygems.org"
git_source(:github) { |repo| "https://github.com/#{repo}.git" }

ruby "3.3.3"

gem "rails", "~> 7.1.3"
gem "sprockets-rails"
gem "puma", "~> 6.4"
gem "jbuilder"
gem "tzinfo-data", platforms: %i[ mingw mswin x64_mingw jruby ]
gem "bootsnap", require: false
gem "sassc-rails"
gem 'typhoeus'
gem "rack-attack"
gem "rack-attack-rate-limit", require: "rack/attack/rate-limit"
gem 'rack-cors'
gem 'rswag-api'
gem 'rswag-ui'
gem 'bootstrap'
gem "nokogiri"

gem "github-markup", require: "github/markup"
gem "redcarpet", :platforms => :ruby
gem "RedCloth"
gem "commonmarker", '0.23.10'
gem "rdoc"
gem "org-ruby"
gem "creole"
gem "wikicloth", github: 'nricciar/wikicloth'
gem "twitter-text"
gem "asciidoctor"
gem "github-linguist"
gem 'rexml'
gem 'appsignal'
gem 'vandamme', github: 'ecosyste-ms/vandamme'
gem "net-pop", github: "ruby/net-pop" # temporary fix for net-pop until ruby 3.3.4

group :development, :test do
  gem "debug", platforms: %i[ mri mingw x64_mingw ]
end

group :development do
  gem "web-console"
end

group :test do
  gem "shoulda"
  gem "webmock"
  gem "mocha"
  gem "rails-controller-testing"
end
