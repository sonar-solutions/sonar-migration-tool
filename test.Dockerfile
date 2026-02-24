FROM python:3.14-slim

RUN apt-get --assume-yes update && apt-get --assume-yes install bash

RUN mkdir /app
COPY requirements.txt /app/requirements.txt

RUN pip install -r /app/requirements.txt

COPY requirements-dev.txt /app/requirements-dev.txt
RUN pip install -r /app/requirements-dev.txt
COPY /src/ /app/

WORKDIR /app/
ENTRYPOINT ["pytest", "/app/tests", "--cov-config=/app/.coveragerc", "--cov=/app", "--cov-report", "xml:/app/files/coverage.xml", "--asyncio-mode", "auto", "-rfE", "-p", "no:warnings", "--log-level", "ERROR"]
