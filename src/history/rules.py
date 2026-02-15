"""Generate active rules protobuf"""
import os
from datetime import datetime, UTC

from protobufs.scanner_report_pb2 import ActiveRule
from history.utilities import append_protobuf_to_file


def generate_active_rules(output_dir, profile_keys, rules):
    """Generate active rules protobuf"""
    for profile in profile_keys.values():
        for rule in rules[profile['language']].values():
            if not rule:
                continue
            active_rule = {
                'rule_repository': rule['rule_repository'],
                'rule_key': rule['key'].split(':')[-1],
                'severity': rule['severity'],
                'createdAt': int(datetime.now(UTC).timestamp()) * 1000,
                'updatedAt': int(datetime.now(UTC).timestamp()) * 1000,
                'q_profile_key': profile['key']
            }
            append_protobuf_to_file(
                file_path=os.path.join(output_dir, 'activerules.pb'),
                proto_class=ActiveRule,
                payload=active_rule,
            )

