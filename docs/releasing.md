# Releasing

We're releasing based on tags, which contain our version number. The format is 'v0.0.0'.
The release title will be set to this version.

The CI pipeline has a 'release' job, which will update the release on Github.
That job triggers itself, when a draft release is created.

## Create new release pipeline

We release from release-branches. Each maintained release has a separate pipeline in Concourse.
To create a new pipeline run this in the CI repository:

```
cd pipelines/cf-operator-release
./configure.sh CFO v0.4.0
```

Where `CFO` is your concourse target and `v0.4.0` is the name of the branch.

## Create a new release

After completion, the pipeline will create several artifacts:

* helm chart on S3
* cf-operator binary on S3
* docker image of the operator on dockerhub

Running the 'release' job will take the latest artificats, which passed through the pipeline and add them to the Github release:

* to the body
* as Github assets for downloading

The version numbers (v0.0.0-build.SHA) of these assets are taken from the info on S3.
The assets will be copied into a 'release' folder on S3.

The docker image is only referenced from the helm chart and not mentioned in the release, though.

## Checklist

0. Create a new release pipeline for a version branch
1. Wait for commit to pass release pipeline
2. Tag commit with new version number
3. Create a draft Github release for that tag, 'release' job triggers
4. Wait for 'release' job to finish on Concourse
5. Edit the draft release on Github and publish it

Try not to push to the pipeline again, until step 4 is completed. The 'release' job will always take the most recent artifacts from S3. Maybe pause the 'publish' job manually to avoid accidental updates.
