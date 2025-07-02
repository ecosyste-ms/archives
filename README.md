# [Ecosyste.ms: Archives](https://archives.ecosyste.ms)

An open API service for inspecting package archives and files from many open source software ecosystems.

## What is Archives?

Archives provides a unified HTTP API to explore the contents of package archives (tarballs, zip files, etc.) from various package registries without needing to download and extract them locally. It acts as a caching proxy that:

- **Lists files** within package archives from npm, PyPI, RubyGems, and other ecosystems
- **Fetches file contents** directly from archives without full downloads
- **Extracts metadata** like READMEs and changelogs from packages
- **Caches responses** to improve performance and reduce load on upstream registries

### Use Cases

- **Security scanning**: Inspect package contents for vulnerabilities without downloading
- **Documentation extraction**: Automatically fetch README files from packages
- **Dependency analysis**: Explore package structures and dependencies
- **Package validation**: Verify package contents match expectations
- **Research**: Analyze package ecosystems at scale

This project is part of [Ecosyste.ms](https://ecosyste.ms): Tools and open datasets to support, sustain, and secure critical digital infrastructure.

## API

Documentation for the REST API is available here: [https://archives.ecosyste.ms/docs](https://archives.ecosyste.ms/docs)

### Quick Start

The API accepts URLs to package archives as parameters. These URLs typically point to:
- npm package tarballs (e.g., from registry.npmjs.org)
- PyPI package wheels/tarballs (e.g., from files.pythonhosted.org)
- RubyGems .gem files (e.g., from rubygems.org)
- Other package archive formats (zip, tar.gz, etc.)

### Example API Calls

#### List files in an archive
```bash
GET /api/v1/archives/list?url=https://registry.npmjs.org/express/-/express-4.18.2.tgz

# Returns: JSON array of file paths in the archive
["package.json", "README.md", "lib/express.js", ...]
```

#### Get contents of a specific file
```bash
GET /api/v1/archives/contents?url=https://registry.npmjs.org/express/-/express-4.18.2.tgz&path=package.json

# Returns: JSON object with file contents
{
  "name": "package.json",
  "directory": false,
  "contents": "{\n  \"name\": \"express\",\n  \"version\": \"4.18.2\",\n  ..."
}
```

#### Extract README
```bash
GET /api/v1/archives/readme?url=https://registry.npmjs.org/express/-/express-4.18.2.tgz

# Returns: README content in multiple formats (raw, HTML, plain text)
```

### Rate Limits

The default rate limit for the API is 5000/req per hour based on your IP address, get in contact if you need to to increase your rate limit.

## Development

For development and deployment documentation, check out [DEVELOPMENT.md](DEVELOPMENT.md)

## Contribute

Please do! The source code is hosted at [GitHub](https://github.com/ecosyste-ms/archives). If you want something, [open an issue](https://github.com/ecosyste-ms/archives/issues/new) or a pull request.

If you need want to contribute but don't know where to start, take a look at the issues tagged as ["Help Wanted"](https://github.com/ecosyste-ms/archives/issues?q=is%3Aopen+is%3Aissue+label%3A%22help+wanted%22).

You can also help triage issues. This can include reproducing bug reports, or asking for vital information such as version numbers or reproduction instructions. 

Finally, this is an open source project. If you would like to become a maintainer, we will consider adding you if you contribute frequently to the project. Feel free to ask.

For other updates, follow the project on Twitter: [@ecosyste_ms](https://twitter.com/ecosyste_ms).

### Note on Patches/Pull Requests

 * Fork the project.
 * Make your feature addition or bug fix.
 * Add tests for it. This is important so we don't break it in a future version unintentionally.
 * Send a pull request. Bonus points for topic branches.

### Vulnerability disclosure

We support and encourage security research on Ecosyste.ms under the terms of our [vulnerability disclosure policy](https://github.com/ecosyste-ms/archives/security/policy).

### Code of Conduct

Please note that this project is released with a [Contributor Code of Conduct](https://github.com/ecosyste-ms/.github/blob/main/CODE_OF_CONDUCT.md). By participating in this project you agree to abide by its terms.

## Maintainers

This project is maintained by the Ecosyste.ms team. You can reach us at:

- **Email**: [support@ecosyste.ms](mailto:support@ecosyste.ms)
- **Twitter**: [@ecosyste_ms](https://twitter.com/ecosyste_ms)
- **GitHub**: [ecosyste-ms](https://github.com/ecosyste-ms)

Project lead: [Andrew Nesbitt](https://github.com/andrew)

## Copyright

Code is licensed under [GNU Affero License](LICENSE) © 2023 [Andrew Nesbitt](https://github.com/andrew).

Data from the API is licensed under [CC BY-SA 4.0](https://creativecommons.org/licenses/by-sa/4.0/).