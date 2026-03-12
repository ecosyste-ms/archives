Rswag::Api.configure do |c|
  c.openapi_root = Rails.root.to_s + '/openapi'
end

# Patch rswag-api middleware to return integer status for Rack 3 compatibility
# https://github.com/rswag/rswag/issues/751
Rswag::Api::Middleware.prepend(Module.new do
  def call(env)
    status, headers, body = super
    [status.to_i, headers, body]
  end
end)
