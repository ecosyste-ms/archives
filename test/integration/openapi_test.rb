require "test_helper"

class OpenapiTest < ActiveSupport::TestCase
  test 'openapi.yaml is valid' do
    f = YAML.load_file(Rails.root.join('openapi/api/v1/openapi.yaml'))
    assert_equal f.class, Hash
  end
end

class OpenapiEndpointTest < ActionDispatch::IntegrationTest
  test 'serves openapi.yaml with integer status' do
    get '/docs/api/v1/openapi.yaml'
    assert_response :success
    assert_includes response.content_type, 'yaml'
  end
end