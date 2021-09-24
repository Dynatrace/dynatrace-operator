FROM python:3.8

LABEL "com.github.actions.name"="Changelog CI"
LABEL "com.github.actions.description"="Changelog CI is a GitHub Action that generates changelog, Then the changelog is committed and/or commented to the release Pull request."
LABEL "com.github.actions.icon"="clock"
LABEL "com.github.actions.color"="blue"

COPY requirements.txt /requirements.txt

RUN pip install -r requirements.txt

COPY changelog-ci.py /changelog-ci.py

RUN ["chmod", "+x", "/changelog-ci.py"]
ENTRYPOINT ["python", "/changelog-ci.py"]
