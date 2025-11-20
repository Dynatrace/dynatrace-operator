import json


supported_environments = [
    "k8s-1.28",
    "k8s-1.29",
    "k8s-1.30",
    "k8s-1.31",
    "k8s-1.32",
    "k8s-1.33",
    "k8s-1.34",
    "ocp-4.12",
    "ocp-4.13",
    "ocp-4.14",
    "ocp-4.15",
    "ocp-4.16",
    "ocp-4.17",
    "ocp-4.18",
]

table_header = [
    {
        "type": "rich_text",
        "elements": [
            {
                "type": "rich_text_section",
                "elements": [{"type": "text", "text": "environment"}],
            }
        ],
    },
    {
        "type": "rich_text",
        "elements": [
            {
                "type": "rich_text_section",
                "elements": [
                    {"type": "text", "text": "result", "style": {"bold": True}}
                ],
            }
        ],
    },
]

single_row_template = [
    {
        "type": "rich_text",
        "elements": [
            {
                "type": "rich_text_section",
                "elements": [{"type": "text", "text": ""}],
            }
        ],
    },
    {
        "type": "rich_text",
        "elements": [
            {
                "type": "rich_text_section",
                "elements": [
                    {
                        "type": "link",
                        "url": "",
                        "text": "tests ",
                    },
                    {"type": "emoji", "name": ""},
                ],
            }
        ],
    },
]

rows = [table_header]

with open("./hack/slack/slack-e2e-ondemand-payload.json", "r+") as f:
    payload = json.loads(f.read())

    for env in supported_environments:
        row = single_row_template.copy()
        row[0]["elements"][0]["elements"][0]["text"] = env
        row[1]["elements"][0]["elements"][0][
            "url"
        ] = "${{{{ env.{}_RUN_ID_URL }}}}".format(
            env.replace("-", "_").replace(".", "_").upper()
        )
        row[1]["elements"][0]["elements"][1]["name"] = "${{{{ env.{}_EMOJI }}}}".format(
            env.replace("-", "_").replace(".", "_").upper()
        )
        rows.append(row)

    payload["blocks"][1] = {
        "type": "table",
        "rows": rows,
    }
    f.seek(0)
    json.dump(payload, f, indent=2)
    f.truncate()
