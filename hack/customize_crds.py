import yaml
from os import listdir
from os.path import isfile, join

crd_directory = "./config/crd/bases/"


def recursively_delete(d, key_of_interest):
    for key, value in list(d.items()):
        if key in key_of_interest:
            print("found and deleted: " + key)
            del d[key]
        if type(value) is dict:
            recursively_delete(value, key_of_interest)


crds = [f for f in listdir(crd_directory) if isfile(join(crd_directory, f))]

for yaml_file in crds:
    with open(crd_directory + yaml_file, 'r') as crd:
        search_string = {"x-kubernetes-int-or-string", "x-kubernetes-list-type", "anyOf", "pattern"}
        file = yaml.full_load(crd)
        recursively_delete(file, search_string)
        del file['status']

    with open(crd_directory + yaml_file, 'w') as new_crd:
        yaml.dump(file, new_crd)
