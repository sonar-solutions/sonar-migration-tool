FROM python:3.14-slim

RUN apt-get --assume-yes update \
    && apt-get --assume-yes install --no-install-recommends bash \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir /app

COPY requirements.txt /app/requirements.txt

RUN pip install -r /app/requirements.txt

COPY /src/ /app/

WORKDIR /app
ENTRYPOINT ["python", "/app/main.py"]
LABEL org.opencontainers.image.source=https://github.com/sonar-solutions/sonar-reports