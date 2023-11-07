import yaml


FILE = '../../config/crd/bases/dynatrace.com_dynakubes.yaml'


def table_header():
    return '|Parameter|Description|Default value|Data type|'


def main():
    with open(FILE, 'r') as file:
        dyna = yaml.safe_load(file)
    spec = dyna['spec']['versions'][1]['schema']['openAPIV3Schema']
    print(spec['description'])
    props = spec['properties']['spec']['properties']
    res = [
        '## {name}'.format(name=spec['description']),
        table_header(),
        '|---|---|---|---|'
    ]
    for prop in props:
        # ignore for now
        if 'properties' not in props[prop]:
            template = '|{field}|{description}|{type}|{default}|'.format(
                field=prop,
                type=props[prop]["type"],
                description=props[prop].get('description', ''),
                default=props[prop].get('default', ''),
            )
            res.append(template)

    print('\n'.join(res) + '\n')


if __name__ == '__main__':
    main()
