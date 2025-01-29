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

## Usage

To test on the command line in a python directory, you can do the following:

```
curl --data @<(echo "{\"head\":\"$(tar -czf - . | base64 -i -)\"}") http://localhost:7777/pull
```

## Deployment

This is also deployed on Google Cloud Run at the following URL via CI:

https://banditize-562949304223.us-central1.run.app/