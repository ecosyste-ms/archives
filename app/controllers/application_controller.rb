class ApplicationController < ActionController::Base
  after_action lambda {
    request.session_options[:skip] = true
  }
end
