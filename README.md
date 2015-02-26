# aci-discover - App Container Image Discovery Server

aci-discover implements the server side of the [App Container Image Discovery protocol][proto].
It hosts App Container images, signatures, and the public GPG keys used to generate those signatures.

[proto]: https://github.com/appc/spec/blob/master/SPEC.md#app-container-image-discovery

Deployment is as simple as placing your ACI files and signatures in `/opt/aci/images/{os}/{arch}/`, your GPG keys at `/opt/aci/pubkeys.gpg` and starting the aci-discover daemon.
For example, to deploy an aci-discover endpoint for `example.com/reduce-worker:0.0.1`, place the
following files on disk and execute `aci-discover --domain=example.com`:

- /opt/aci/images/linux/amd64/reduce-worker-0.0.1.aci
- /opt/aci/images/linux/amd64/reduce-worker-0.0.1.sig
- /opt/aci/pubkeys.gpg

## GPG

The App Container specification encourages the use of GPG signatures to verify the integrity of image data.

Generate the required `pubkeys.gpg` file using a command like this:

```
gpg --armor --output /opt/aci/pubkeys.gpg --export
```

A detached GPG signature could be generated using the following command:

```
gpg --armor --output /opt/aci/images/linux/amd64/reduce-worker-0.0.1.sig \
  --detach-sig /opt/aci/images/linux/amd64/reduce-worker-0.0.1.aci
```

## TODO

- support for storage of image data in cloud services (e.g. Google Cloud Storage, Amazon S3, etc)
