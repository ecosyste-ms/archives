class HomeController < ApplicationController
  def index
    @formats = [
      {
        name: '.tgz/.tar.gz',
        ecosystems: ['npm', 'pub', 'cran', 'hackage', 'puppet']
      },
      {
        name: '.zip',
        ecosystems: ['go', 'elm']
      },
      {
        name: '.tar',
        ecosystems: ['hex']
      },
      {
        name: '.gem',
        ecosystems: ['rubygems']
      },
      {
        name: '.nupkg',
        ecosystems: ['nuget']
      },
      {
        name: '.crate',
        ecosystems: ['cargo']
      },
    ]
  end
end