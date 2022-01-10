#!/usr/bin/env python3

import sys

out = ""

current_section = ""
section_started = False
empty_section = True
for line in sys.stdin:
    if line.startswith("---"):
        section_started = not section_started
    elif not line.startswith("#") and line != "\n" and line != "":
        empty_section = False
    if not section_started:
        if not empty_section:
            current_section += line
            out += current_section
        current_section = ""
        empty_section = True
        section_started = True
    else:
        current_section += line

out += current_section
sys.stdout.write(out)
