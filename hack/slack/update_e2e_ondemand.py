import json
from ruamel.yaml import YAML

ONDEMAND_FILE = "./.github/workflows/e2e-tests-ondemand.yaml"


yaml = YAML()
yaml.width = 4096

# read ondemand_file file to dict and update
with open(ONDEMAND_FILE, "r") as f:
    data = yaml.load(f)

# extract matrix include to dynamically generate slack message table rows
matrix = data["jobs"]["run-matrix"]["strategy"]["matrix"]["include"]


supported_k8s = sorted(
    [
        f"{item['platform']}_{item['version']}"
        for item in matrix
        if item["platform"] == "k8s"
    ]
)
supported_ocps = sorted(
    [
        f"{item['platform']}_{item['version']}"
        for item in matrix
        if item["platform"] == "ocp"
    ]
)


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

title_section = {
    "type": "section",
    "text": {
        "type": "mrkdwn",
        "text": "E2E ondemand test results for `${{ github.sha }}` ",
    },
}

footer_section = {
    "type": "rich_text",
    "elements": [
        {
            "type": "rich_text_section",
            "elements": [
                {
                    "type": "link",
                    "text": "detailed logs of run #${{ github.runNumber }}",
                    "url": "https://github.com/${{ github.payload.repository.full_name }}/actions/runs/${{ github.runId }}",
                }
            ],
        }
    ],
}

base = {"blocks": []}

table_section = {"type": "table", "rows": []}

rows = [table_header]


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
                            "url": "${{{{ env.{}_RUN_ID_URL }}}}".format(k8s_env_clean),
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
                            "url": "${{{{ env.{}_RUN_ID_URL }}}}".format(ocp_env_clean),
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

table_section["rows"] = rows

base["blocks"].extend([title_section, table_section, footer_section])

print(json.dumps(base, indent=2))
