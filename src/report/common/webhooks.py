from collections import defaultdict

from parser import extract_path_value
from report.utils import generate_section
from utils import multi_extract_object_reader
from datetime import datetime


def process_webhook(webhooks, webhook, server_id):
    webhooks[server_id][webhook['name']] = dict(
        server_id=server_id,
        name=extract_path_value(obj=webhook, path='$.name'),
        url=extract_path_value(obj=webhook, path='$.url'),
        project=extract_path_value(obj=webhook, path='$.projectKey'),
        has_secret=extract_path_value(obj=webhook, path='$.hasSecret'),
        deliveries=0,
        successes=0,
        failures=0,
        last_success_date=None,
        last_success=None,
        last_error_date=None,
        last_error=None,
    )
    return webhooks


def process_delivery(webhooks, server_id, delivery):
    name = extract_path_value(obj=delivery, path='$.name')
    if name not in webhooks[server_id].keys():
        return webhooks
    webhook = webhooks[server_id][name]
    webhook['deliveries'] += 1
    delivery_date = extract_path_value(obj=delivery, path='$.at')
    try:
        delivery_date = datetime.strptime(delivery_date, '%Y-%m-%dT%H:%M:%S%z')
    except (ValueError, TypeError):
        return webhooks
    if delivery['success']:
        webhook['successes'] += 1
        webhook['last_success'] = max(webhooks['last_success'], delivery_date)
    else:
        webhook['failures'] += 1
        webhook['last_error'] = max(webhooks['last_error_date'], delivery_date)
    return webhooks


def process_webhooks(directory, extract_mapping, server_id_mapping):
    webhooks = defaultdict(dict)
    for key in ['getWebhooks', 'getProjectWebhooks']:
        for url, webhook in multi_extract_object_reader(directory=directory, mapping=extract_mapping, key=key):
            server_id = server_id_mapping[url]
            webhooks = process_webhook(webhooks=webhooks, webhook=webhook, server_id=server_id)
    for key in ['getWebhookDeliveries', 'getProjectWebhookDeliveries']:
        for url, delivery in multi_extract_object_reader(directory=directory, mapping=extract_mapping, key=key):
            server_id = server_id_mapping[url]
            webhooks = process_delivery(webhooks=webhooks, delivery=delivery, server_id=server_id)
    return webhooks

def generate_webhook_markdown(directory, extract_mapping, server_id_mapping):
    webhooks = process_webhooks(directory=directory, extract_mapping=extract_mapping, server_id_mapping=server_id_mapping)
    return generate_section(
        title='Webhooks',
        headers_mapping={
            "Server ID": "server_id",
            "Webhook Name": "name",
            "URL": "url",
            "Project": "project",
            "Deliveries": "deliveries",
            "Successful Deliveries": "successes",
            "Failed Deliveries": "failures",
            "Last Successful Delivery": "last_success",
            "Last Failed Delivery": "last_error",
        },
        rows=[webhook for server_webhooks in webhooks.values() for webhook in server_webhooks.values()],
    )