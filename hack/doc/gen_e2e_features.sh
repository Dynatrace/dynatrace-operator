#!/bin/sh
gomarkdoc_header="<!-- Code generated by gomarkdoc. DO NOT EDIT -->"
script_header="<!-- Code generated by ./hack/doc/gen_e2e_features.sh - DO NOT EDIT -->"
doc_dir="./test/features"
output_dir="./doc/e2e"
output_file="features.md"
output=""

# get dirs containing doc packages
doc_dir_subdirs=$(find $doc_dir -type d)

# order dirs alphabetically
doc_dir_subdirs=$(echo "$doc_dir_subdirs" | sort)

# append all gomarkdoc outputs in a single variable
for dir in $doc_dir_subdirs; do
  if [ "$dir" != "$doc_dir" ]; then
    output="${output}$(GOARCH="e2e" gomarkdoc --repository.url "https://github.com/Dynatrace/dynatrace-operator" --repository.path "/" --repository.default-branch "main" "${dir}" | sed 's/\\//g')"
    # remove gomarkdoc footer
    output=$(echo "${output}" | sed '$d')
  fi
done

# remove gomarkdoc headers and add custom one
output=$(echo "${output}" | sed s/"$gomarkdoc_header"//)
output="${script_header}
${output}"

# write output to file
mkdir -p $output_dir
echo "$output" > $output_dir/$output_file

# fix linting issues
markdownlint -f $output_dir/$output_file
