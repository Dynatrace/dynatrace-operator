import json


supported_k8s = [
    "k8s-1.28",
    "k8s-1.29",
    "k8s-1.30",
    "k8s-1.31",
    "k8s-1.32",
    "k8s-1.33",
    "k8s-1.34",
]

supported_ocps = [
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


rows = [table_header]

with open("./hack/slack/slack-e2e-ondemand-payload.json") as f:
    payload = json.load(f)


with open("./hack/slack/slack-e2e-ondemand-payload.json", "w") as f1:
    for k8s_env, ocp_env in zip(supported_k8s, supported_ocps):
        k8s_env_clean = k8s_env.replace("-", "_").replace(".", "_").upper()
        ocp_env_clean = ocp_env.replace("-", "_").replace(".", "_").upper()
        row = [
            {
                "type": "rich_text",
                "elements": [
                    {
                        "type": "rich_text_section",
                        "elements": [{"type": "text", "text": k8s_env}],
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
                                "url": "${{{{ env.{}_RUN_ID_URL }}}}".format(
                                    k8s_env_clean
                                ),
                                "text": "tests ",
                            },
                            {
                                "type": "emoji",
                                "name": "${{{{ env.{}_EMOJI }}}}".format(k8s_env_clean),
                            },
                        ],
                    }
                ],
            },
            {
                "type": "rich_text",
                "elements": [
                    {
                        "type": "rich_text_section",
                        "elements": [{"type": "text", "text": ocp_env}],
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
                                "url": "${{{{ env.{}_RUN_ID_URL }}}}".format(
                                    ocp_env_clean
                                ),
                                "text": "tests ",
                            },
                            {
                                "type": "emoji",
                                "name": "${{{{ env.{}_EMOJI }}}}".format(ocp_env_clean),
                            },
                        ],
                    }
                ],
            },
        ]

        rows.append(row)

    payload["blocks"][1] = {
        "type": "table",
        "rows": rows,
    }
    json.dump(payload, f1, indent=2)
