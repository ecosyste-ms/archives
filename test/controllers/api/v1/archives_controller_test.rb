require 'test_helper'

class ApiV1ArchivesControllerTest < ActionDispatch::IntegrationTest
  test 'list' do
    stub_request(:get, "https://registry.npmjs.org/base62/-/base62-2.0.1.tgz")
      .to_return({ status: 200, body: File.open(File.join(Rails.root, 'test', 'fixtures', 'files','base62-2.0.1.tgz')).read })

    get list_api_v1_archives_path(url: 'https://registry.npmjs.org/base62/-/base62-2.0.1.tgz')
    assert_response :success
    actual_response = JSON.parse(@response.body)

    assert_equal actual_response, [
      ".codeclimate.yml",
      ".eslintignore",
      ".eslintrc",
      ".travis.yml",
      "CODE_OF_CONDUCT.md",
      "CONTRIBUTING.md",
      "LICENSE",
      "Readme.md",
      "benchmark",
      "benchmark/benchmarks.js",
      "benchmark/benchmarks_legacy.js",
      "fork",
      "fork/.editorconfig",
      "fork/.eslintrc",
      "fork/README.md",
      "fork/package.json",
      "fork/src",
      "fork/src/ascii.js",
      "fork/src/custom.js",
      "fork/test",
      "fork/test/test_base62_ascii.js",
      "fork/test/test_base62_custom.js",
      "index.d.ts",
      "lib",
      "lib/ascii.js",
      "lib/custom.js",
      "lib/legacy.js",
      "package.json", 
      "test",
      "test/test_ascii.js",
      "test/test_custom.js",
      "test/test_legacy.js"
    ]
  end

  test 'list zip' do
    stub_request(:get, "https://github.com/adobe/parcel-plugin-htl/archive/refs/heads/master.zip")
      .to_return({ status: 200, body: File.open(File.join(Rails.root, 'test', 'fixtures', 'files','parcel-plugin-htl-master.zip')).read })

    get list_api_v1_archives_path(url: 'https://github.com/adobe/parcel-plugin-htl/archive/refs/heads/master.zip')
    assert_response :success
    actual_response = JSON.parse(@response.body)

    assert_equal actual_response, [".circleci",
      ".circleci/config.yml",
      ".eslintignore",
      ".eslintrc.js",
      ".github",
      ".github/move.yml",
      ".gitignore",
      ".npmignore",
      ".releaserc.js",
      ".snyk",
      "CHANGELOG.md",
      "CODE_OF_CONDUCT.md",
      "CONTRIBUTING.md",
      "LICENSE.txt",
      "README.md",
      "package-lock.json",
      "package.json",
      "src",
      "src/HTLAsset.js",
      "src/HelixJSAsset.js",
      "src/engine",
      "src/engine/RuntimeTemplate.js",
      "src/index.js",
      "test",
      "test/example",
      "test/example/bla.css",
      "test/example/html.htl",
      "test/testGeneratedCode.js"
    ]
  end

  test 'contents of a file' do
    stub_request(:get, "https://registry.npmjs.org/base62/-/base62-2.0.1.tgz")
      .to_return({ status: 200, body: File.open(File.join(Rails.root, 'test', 'fixtures', 'files','base62-2.0.1.tgz')).read })

    get contents_api_v1_archives_path(url: 'https://registry.npmjs.org/base62/-/base62-2.0.1.tgz', path: '.eslintignore')
    assert_response :success
    actual_response = JSON.parse(@response.body)

    assert_equal actual_response, {
      "name"=>".eslintignore", 
      "directory"=>false, 
      "contents"=>"**/*{.,-}min.js
"
}
  end

  test 'contents of a folder' do
    stub_request(:get, "https://registry.npmjs.org/base62/-/base62-2.0.1.tgz")
      .to_return({ status: 200, body: File.open(File.join(Rails.root, 'test', 'fixtures', 'files','base62-2.0.1.tgz')).read })

    get contents_api_v1_archives_path(url: 'https://registry.npmjs.org/base62/-/base62-2.0.1.tgz', path: 'lib')
    assert_response :success
    actual_response = JSON.parse(@response.body)

    assert_equal actual_response, {
      "name"=>"lib",
      "directory"=>true,
      "contents"=>["ascii.js", "custom.js", "legacy.js"]
    }
  end

  test 'contents of a missing path' do
    stub_request(:get, "https://registry.npmjs.org/base62/-/base62-2.0.1.tgz")
      .to_return({ status: 200, body: File.open(File.join(Rails.root, 'test', 'fixtures', 'files','base62-2.0.1.tgz')).read })

    get contents_api_v1_archives_path(url: 'https://registry.npmjs.org/base62/-/base62-2.0.1.tgz', path: 'fib')
    assert_response :missing
  end
end