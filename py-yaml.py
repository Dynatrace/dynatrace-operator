#!/usr/bin/env python

import yaml


with open("config/helm/chart/default/Chart.yaml", "r") as file:
    try:
        print(yaml.safe_load(file))
    except yaml.YAMLError as exc:
        print(exc)

print()

with open("config/helm/chart/default/Chart.yaml", "r") as file:
    try:
        print(yaml.dump(yaml.safe_load(file), default_flow_style=False))        
    except yaml.YAMLError as exc:
        print(exc)
