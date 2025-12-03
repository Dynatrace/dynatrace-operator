## Runs markdownlint using existing .markdownlint.json config file through all .md files in the project
markdown/lint: prerequisites/markdownlint
	# --disable MD034 MD037 - workaround for errors in k8s.io/api package (type PersistentVolumeClaimSpec)
	$(MARKDOWNLINT) --ignore node_modules --disable MD034 MD037 -- .

## Runs markdown-link-check for all .md files in the project
markdown/link-check: prerequisites/markdown-link-check
	$(MARKDOWN_LINK_CHECK) --ignore node_modules,.git,testdata .
