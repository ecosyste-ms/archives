# Development

## Setup

First things first, you'll need to fork and clone the repository to your local machine.

`git clone https://github.com/ecosyste-ms/archives.git`

The project is written in Go and requires:

- [Go 1.24+](https://go.dev/dl/)
- [Node.js 16+](https://nodejs.org/en/download/) (for repomix)

Optional tools for rendering non-Markdown README formats:

- [asciidoctor](https://asciidoctor.org/) for AsciiDoc
- [docutils](https://docutils.sourceforge.io/) (rst2html) for reStructuredText
- [pandoc](https://pandoc.org/) for Textile, Org, Creole, MediaWiki
- [perl](https://www.perl.org/) (pod2html) for Pod

Once you've got Go installed, from the root directory of the project run:

```
go run ./cmd/server/
```

You can then load up [http://localhost:5000](http://localhost:5000) to access the service.

### Docker

Alternatively you can use the existing docker configuration files to run the app in a container.

Run this command from the root directory of the project to start the service.

`docker-compose up --build`

You can then load up [http://localhost:5000](http://localhost:5000) to access the service.

## Tests

Run all tests with:

`go test ./...`

Run benchmarks with:

`go test ./internal/archive/ -bench=. -benchmem`

## Deployment

A container-based deployment is highly recommended, we use [dokku.com](https://dokku.com/).
