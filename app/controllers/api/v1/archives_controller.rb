class Api::V1::ArchivesController < Api::V1::ApplicationController
  def list
    expires_in(60.days, public: true, "s-maxage" => 60.days) # TODO this needs to be more dynamic to take into account headers from where the file was loaded
    @archive = Archive.new(params[:url])
    render json: @archive.list_files
  end

  def contents
    expires_in(60.days, public: true, "s-maxage" => 60.days) # TODO this needs to be more dynamic to take into account headers from where the file was loaded
    @archive = Archive.new(params[:url])
    contents = @archive.contents(params[:path])
    if contents.nil?
      render json: {:error => "path not found"}, :status => 404
    else
      render json: contents
    end
  end

  def readme
    expires_in(60.days, public: true, "s-maxage" => 60.days) # TODO this needs to be more dynamic to take into account headers from where the file was loaded
    @archive = Archive.new(params[:url])
    readme = @archive.readme
    if readme.nil?
      render json: {:error => "path not found"}, :status => 404
    else
      render json: readme
    end
  end

  def changelog
    expires_in(60.days, public: true, "s-maxage" => 60.days) # TODO this needs to be more dynamic to take into account headers from where the file was loaded
    @archive = Archive.new(params[:url])
    changelog = @archive.changelog
    if changelog.nil?
      render json: {:error => "path not found"}, :status => 404
    else
      render json: changelog
    end
  end

  def repopack
    expires_in(60.days, public: true, "s-maxage" => 60.days) # TODO this needs to be more dynamic to take into account headers from where the file was loaded
    @archive = Archive.new(params[:url])
    repopack = @archive.repopack
    if repopack.nil?
      render json: {:error => "path not found"}, :status => 404
    else
      render json: repopack
    end
  end
end