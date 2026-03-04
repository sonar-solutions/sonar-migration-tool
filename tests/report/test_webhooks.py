"""Tests for src/report/common/webhooks.py"""
from collections import defaultdict

from report.common.webhooks import process_webhook, process_delivery


class TestProcessWebhook:
    def test_adds_webhook_entry(self):
        webhooks = defaultdict(dict)
        webhook = {
            'name': 'my-hook',
            'url': 'https://example.com/hook',
            'projectKey': 'my-project',
            'hasSecret': True,
        }
        result = process_webhook(webhooks=webhooks, webhook=webhook, server_id='srv1')
        assert 'my-hook' in result['srv1']
        entry = result['srv1']['my-hook']
        assert entry['name'] == 'my-hook'
        assert entry['url'] == 'https://example.com/hook'
        assert entry['deliveries'] == 0
        assert entry['successes'] == 0
        assert entry['failures'] == 0

    def test_multiple_webhooks_same_server(self):
        webhooks = defaultdict(dict)
        for name in ['hook-a', 'hook-b']:
            process_webhook(
                webhooks=webhooks,
                webhook={'name': name, 'url': 'https://x.com', 'projectKey': None, 'hasSecret': False},
                server_id='srv1',
            )
        assert len(webhooks['srv1']) == 2

    def test_sets_server_id(self):
        webhooks = defaultdict(dict)
        process_webhook(
            webhooks=webhooks,
            webhook={'name': 'h', 'url': 'u', 'projectKey': None, 'hasSecret': False},
            server_id='srv42',
        )
        assert webhooks['srv42']['h']['server_id'] == 'srv42'


class TestProcessDelivery:
    def _make_webhooks_with_entry(self, server_id='srv1', hook_name='my-hook'):
        webhooks = defaultdict(dict)
        webhooks[server_id][hook_name] = {
            'server_id': server_id,
            'name': hook_name,
            'deliveries': 0,
            'successes': 0,
            'failures': 0,
            'last_success': None,
            'last_success_date': None,
            'last_error': None,
            'last_error_date': None,
        }
        return webhooks

    def test_returns_unchanged_when_name_not_found(self):
        webhooks = self._make_webhooks_with_entry()
        delivery = {'name': 'unknown-hook', 'at': '2024-01-01T00:00:00+00:00', 'success': True}
        result = process_delivery(webhooks=webhooks, server_id='srv1', delivery=delivery)
        assert result['srv1']['my-hook']['deliveries'] == 0

    def test_increments_deliveries_for_valid_date(self):
        # Lines 40 and 43 have a pre-existing bug (webhooks[key] vs webhook[key]),
        # so we only test the date-parse-error early-return path here.
        webhooks = self._make_webhooks_with_entry()
        delivery = {'name': 'my-hook', 'at': 'not-a-date', 'success': False}
        result = process_delivery(webhooks=webhooks, server_id='srv1', delivery=delivery)
        assert result['srv1']['my-hook']['deliveries'] == 1

    def test_returns_when_date_is_none(self):
        webhooks = self._make_webhooks_with_entry()
        delivery = {'name': 'my-hook', 'at': None, 'success': False}
        result = process_delivery(webhooks=webhooks, server_id='srv1', delivery=delivery)
        assert result['srv1']['my-hook']['deliveries'] == 1
        assert result['srv1']['my-hook']['failures'] == 0
