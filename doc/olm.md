# How To Test OLM

If you want to know what is actually happening in the background before testing, go to the [What is actually happening](#what-is-actually-happening) section.

Prerequisites:

- [operator-sdk v1.36.0](https://sdk.operatorframework.io/docs/installation/)
- docker or podman
- OLM enabled cluster. Either use Openshift or install manually: https://operatorhub.io/how-to-install-an-operator

The basic flow is as follows:

- Generate bundle: `make bundle`
- Build catalog image: `make bundle/build`
- Push catalog image: `make bundle/push`
- Install the bundle: `make bundle/install`
- Run tests
- Clean up bundle: `make bundle/cleanup`

Install depends on build and push so building and deploying can be abbreviated to: `make bundle bundle/install`

The bundle generate can be configured using the following environment variables:

- `PLATFORM`: Target platform. Must be either `openshift` (default) or `kubernetes`
- `VERSION`: Version of the bundle. Does not affect the deployed operator version
- `BUNDLE_IMG`: Override for the bundle image.
- `CHANNELS`: Set bundle channels
- `DEFAULT_CHANNEL`: Set bundle default channel

Example:

```sh
make images/build/push REGISTRY=quay.io
make bundle bundle/install REGISTRY=quay.io PLATFORM=kubernetes
make bundle bundle/upgrade REGISTRY=quay.io PLATFORM=kubernetes VERSION=0.0.2
make bundle/cleanup
```

## How to test an upgrade

> [!IMPORTANT]
> When installing on Openshift, check the `replaces` field in the `ClusterServiceVersion` so that the upgrade actually happens.

1. Deploy the version you are upgrading from. (follow the steps above)
2. Run `make bundle/upgrade`
3. Look at cluster, monitor if the upgrade is successful.

## What is actually happening

> [!NOTE]
> This is only a quick and simple explanation more detailed description check the [olm-docs](https://olm.operatorframework.io/docs/tasks/creating-a-catalog/).

First lets understand each part we are creating.

- What is a `bundle`:
  - We generate this from the manifests (CRD included)
  - Can be considered as an *"OLM Release"*
  - It creates the following file structure
    - `manifests` contains the manfests that will be deployed by OLM. Most of our *normal* manifests are moved into the `ClusterServiceVersion`.
    - `metadata` contains some metadata used by OLM, not very interesting
    - `bundle.Dockerfile` is used to build the catalog container image.

```sh
  config/olm/{PLATFORM}/${VERSION}
      |
      |---- manifests/
      |
      |---- metadata/
      |
      |---- bundle.Dockerfile
```

- What is a `catalog`:
  - A container image that contains the contents of a bundle, + is labeled with the metadata of the bundle.
  - We put it in an `index` to be accessed by OLM.

- What is an `index`:
  - A container image that contains a set of `catalogs`.
  - Created/managed by the `opm` CLI tool
  - Referenced in the `CatalogSource` resources, which is used by OLM to list what can be installed and do the upgrades when possible.
