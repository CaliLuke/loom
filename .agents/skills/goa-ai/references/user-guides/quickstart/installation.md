# Goa-AI Quickstart — Installation

## Prerequisites

Go 1.24+

```bash
go version
```

Install the Goa CLI:

```bash
go install goa.design/goa/v3/cmd/goa@latest
goa version
```

## Project setup

```bash
mkdir quickstart && cd quickstart
go mod init quickstart
go get goa.design/goa/v3@latest goa.design/goa-ai@latest
```
