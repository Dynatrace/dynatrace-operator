#!/usr/bin/python
import re


def sed(path, pattern, replacement):
    patternRegex = re.compile(pattern)

    with open(path + '.output', 'w') as output:
        with open(path, 'r') as source:
            for line in source:
                output.write(patternRegex.sub(replacement, line))

    #shutil.copystat(path, path + '.output')
    #shutil.move(path + '.output', path)

sed('file.txt', r'^version: .*', 'version: 9')

