# Goa-AI Quickstart — Installation

## Prerequisites

Go 1.24+

```bash
go version
```

Install the Goa CLI:

```bash
go install github.com/CaliLuke/loom/cmd/goa@latest
goa version
```

## Project setup

```bash
mkdir quickstart && cd quickstart
go mod init quickstart
go get github.com/CaliLuke/loom@latest github.com/CaliLuke/loom-mcp@latest
```
