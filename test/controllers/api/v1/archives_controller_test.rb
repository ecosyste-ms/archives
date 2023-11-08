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

  test 'list jar' do
    stub_request(:get, "https://repo.clojars.org/org/clojars/majorcluster/clj-data-adapter/0.2.1/clj-data-adapter-0.2.1.jar")
      .to_return({ status: 200, body: File.open(File.join(Rails.root, 'test', 'fixtures', 'files','clj-data-adapter-0.2.1.jar')).read })

    get list_api_v1_archives_path(url: 'https://repo.clojars.org/org/clojars/majorcluster/clj-data-adapter/0.2.1/clj-data-adapter-0.2.1.jar')
    assert_response :success
    actual_response = JSON.parse(@response.body)

    assert_equal actual_response, ["MANIFEST.MF",
      "core.clj",
      "leiningen",
      "leiningen/org.clojars.majorcluster",
      "leiningen/org.clojars.majorcluster/clj-data-adapter",
      "leiningen/org.clojars.majorcluster/clj-data-adapter/README.md",
      "leiningen/org.clojars.majorcluster/clj-data-adapter/project.clj",
      "maven",
      "maven/org.clojars.majorcluster",
      "maven/org.clojars.majorcluster/clj-data-adapter",
      "maven/org.clojars.majorcluster/clj-data-adapter/pom.properties",
      "maven/org.clojars.majorcluster/clj-data-adapter/pom.xml"
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

  test 'readme' do
    stub_request(:get, "https://registry.npmjs.org/base62/-/base62-2.0.1.tgz")
      .to_return({ status: 200, body: File.open(File.join(Rails.root, 'test', 'fixtures', 'files','base62-2.0.1.tgz')).read })

    get readme_api_v1_archives_path(url: 'https://registry.npmjs.org/base62/-/base62-2.0.1.tgz')
    assert_response :success
    actual_response = JSON.parse(@response.body)

    assert_equal actual_response['name'], 'Readme.md'
    assert_equal actual_response['raw'][0..30], "# [Base62.js](http://libraries."
    assert_equal actual_response['html'][0..30], "<h1><a href=\"http://libraries.i"
    assert_equal actual_response['plain'][0..8], "Base62.js"
    
    assert_equal actual_response['extension'], '.md'
    assert_equal actual_response['language'], "Markdown"
    assert_equal actual_response['other_readme_files'], []
  end

  test 'changelog' do
    stub_request(:get, "https://github.com/splitrb/split/archive/refs/heads/main.zip")
      .to_return({ status: 200, body: File.open(File.join(Rails.root, 'test', 'fixtures', 'files','main.zip')).read })

    get changelog_api_v1_archives_path(url: 'https://github.com/splitrb/split/archive/refs/heads/main.zip')
    assert_response :success
    actual_response = JSON.parse(@response.body)

    assert_equal actual_response['name'], 'CHANGELOG.md'
    assert_equal actual_response['raw'][0..20], "# 4.0.2 (December 2nd"
    assert_equal actual_response['html'][0..30], "<h1>4.0.2 (December 2nd, 2022)<"
    assert_equal actual_response['plain'][0..8], "4.0.2 (De"
    
    assert_equal actual_response['extension'], '.md'
    assert_equal actual_response['language'], "Markdown"
  end
end