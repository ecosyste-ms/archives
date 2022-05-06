class Api::V1::ArchivesController < Api::V1::ApplicationController
  def list
    expires_in(60.days, public: true, "s-maxage" => 60.days) # TODO this needs to be more dynamic to take into account headers from where the file was loaded
    @archive = Archive.new(params[:url])
    render json: @archive.list_files
  end

  def contents
    expires_in(60.days, public: true, "s-maxage" => 60.days) # TODO this needs to be more dynamic to take into account headers from where the file was loaded
    @archive = Archive.new(params[:url])
    render plain: @archive.file_contents(params[:path])
  end
end