# banditize
PyCQA/bandit as a web service.

This is a small web service (designed to be run under [Google Cloud Run](https://cloud.google.com/run) or the like that wraps [PyCQA/bandit](https://github.com/PyCQA/bandit) in a JSON-based API.  The input format is:

```json
{
    "base": "... base64-encoded .tar.gz of the base filesystem ...",
    "head": "... base64-encoded .tar.gz of the proposed changed filesystem ..."
}
```

You may omit `base` to scan a directory from scratch.

In turn, this currently returns a single field, but we may expand it later:

```json
{
    "simpleFindings": "... bandit output ..."
}

## Deployment

This has been built (on a Mac) using:

```shell
GOOS=linux GOARCH=amd64 go build . && docker build --platform linux/amd64 . -t us-east4-docker.pkg.dev/minder-zoo/banditize/banditize@sha256:latest
```

And then deployed on Cloud Run at the following URL:

https://banditize-562949304223.us-central1.run.app/