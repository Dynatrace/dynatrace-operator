## How to use [./update_e2e_ondemand.py](update_e2e_ondemand.py)

### Pre-requisites

- python3
- ruamel.yaml

Activate virtual environment:
> [!NOTE]
> We use the same virtual environment as other hack scripts

```bash
source bin/.venv/bin/activate
```

or create a new one:

```bash
# local dev repo - direct call
# Create python virtual env
python3 -m venv venv
# activate virtual env
source venv/bin/activate
pip install pyyaml
```

Run script to generate e2e on-demand Slack payload:

```bash
python3 hack/slack/update_e2e_ondemand.py > hack/slack/slack-e2e-ondemand-payload.json
```

### How to validate payloads using Slack's Block Kit Builder

Open [Block Kit Builder](https://app.slack.com/block-kit-builder/)
and paste the content of `hack/slack/slack-e2e-ondemand-payload.json` into the left panel
