#!/usr/bin/env bash

input="$1"
output="$2"

license='# Copyright 2021 Dynatrace LLC

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#     http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.'

echo '{{- $platformIsSet := printf "%s" (required "Platform needs to be set to kubernetes, openshift, google" (include "dynatrace-operator.platformSet" .))}}' > "$output"
echo '{{ if eq (include "dynatrace-operator.partial" .) "false" }}'   >> "$output"
echo "$license"  													                            >> "$output"
echo '' 																		                          >> "$output"
cat  "$input"					                                                >> "$output"
echo '' 																		                          >> "$output"
echo '  {{- end -}}' 															                    >> "$output"

