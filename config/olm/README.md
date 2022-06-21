# How To Test OLM

If you want to know what is actually happening in the background before testing, go to the `What is actually happening` section.

Prerequisites:
- [operator-sdk](https://sdk.operatorframework.io/docs/installation/)
- [opm](https://docs.openshift.com/container-platform/4.6/cli_reference/opm-cli.html)
- docker or podman


Steps:
1. Get an Openshift cluster that has OLM, and use `oc login` on it
2. If want you want to test but there is no bundle yet => run `make bundle`
   - example: `make bundle PLATFORM=openshift VERSION=0.6.0`
3. Run `make test/olm` (with the same args as the `make bundle` + TAG)
   - example: `make test/olm PLATFORM=openshift VERSION=0.6.0 TAG=test`
4. Go to the UI of the Openshift cluster
   - Go to `Operators -> OperatorHub`
   - Search for `Dynatrace`
   - Find your test operator among the actually released ones and install it.
5. (If you don't want to use the UI) set `CREATE_SUBSCRIPTION=true`, this should create the subscription to deploy the operator.
   - (the UI approach is more reliable)

## How to test an upgrade
**IMPORTANT: Check the `replaces` field in the `ClusterServiceVersion` so that the upgrade actually happens**

Testing an upgrade can't be fully automated (it would be very flaky).

So the basic workflow is:
1. Deploy the version you are upgrading from. (follow the steps above)
2. **(if it doesn't exist)** Create a the bundle for the version you want to upgrade to.
3. In `hack/setup_olm_catalog.sh` there is commented out line, READ IT and update the script accordingly.
   - You are basically creating an index that has two catalog in it. One referencing the old and one the new version.
   - Then updating the `CatalogSource` on the cluster to use the index with the 2 entries, which will cause an upgrade.
4. Run `make test-olm` (with the same args as the `make bundle` + TAG)
5. Look at cluster, monitor if the upgrade is successful.


## What is actually happening
**DISCLAIMER: This is only a quick and simple explanation more detailed description [here](https://olm.operatorframework.io/docs/tasks/creating-a-catalog/)**

First lets understand each part we are creating.

- What is a `bundle`:
  - We generate this from the manifests (crd included)
  - Can be considered as an "OLM Release"
  - It creates the following file structure
  ```
  config/olm/{PLATFORM}/
      |
      |----- {VERSION}/
      |          |
      |          |---- manifests/
      |          |
      |          |---- metadata/
      |
      |
      |---- bundle-{VERSION}.Dockerfile
  ```
   - `manifests` contains all the Kubernetes / Openshift yamls that will be deployed by OLM
      - Most of our *normal* yamls are moved into the `ClusterServiceVersion`.
   - `metadata` contains some metadata used by OLM, not very interesting
   - `bundle-{VERSION}.Dockerfile` is the docker file for creating the `catalog` image for the bundle/release.

- What is a `catalog`:
  - A container image that contains the contents of a bundle, + is labeled with the metadata of the bundle.
  - We put it in an `index` to be accessed by OLM.

- What is an `index`:
  - A container image that contains a set of `catalogs`.
  - Created/managed by the `opm` CLI tool
  - Referenced in the `CatalogSource` resources, which is used by OLM to list what can be installed and do the upgrades when possible.

So `hack/setup_olm_catalog.sh` (used by `make test/olm`) does the following:
1. Creates/pushes `catalog` image for specified bundle
2. Creates/pushes `index` image (by adding the `catalog` to it)
   - can add a new `catalog` to an existing `index` which creates a new `index`
3. Creates a `CatalogSource` in the cluster, referencing the `index` previous pushed
4. (optional) Creates a `Subscription` referencing the `CatalogSource`, which causes OLM to deploy the operator.
