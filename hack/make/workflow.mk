## Trigger workflow related scripts
.PHONY:
workflow: workflow/bump-release-branch

## Generate API docs for custom resources
workflow/bump-release-branch: prerequisites/python
	source local/.venv/bin/activate && python3 ./hack/ci/update-e2e-ondemand-pipeline.py .github/workflows/e2e-tests-ondemand.yaml .github/workflows/e2e-tests-ondemand.yaml
