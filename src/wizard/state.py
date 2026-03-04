from enum import Enum
from dataclasses import dataclass
import json
import os


class WizardPhase(Enum):
    INIT = "init"
    EXTRACT = "extract"
    STRUCTURE = "structure"
    ORG_MAPPING = "org_mapping"
    MAPPINGS = "mappings"
    VALIDATE = "validate"
    MIGRATE = "migrate"
    PIPELINES = "pipelines"
    COMPLETE = "complete"


@dataclass
class WizardState:
    phase: WizardPhase
    extract_id: str | None = None
    source_url: str | None = None
    target_url: str | None = None
    enterprise_key: str | None = None
    organizations_mapped: bool = False
    validation_passed: bool = False
    migration_run_id: str | None = None

    def save(self, directory: str):
        """Persist state to disk for resume capability"""
        state_file = os.path.join(directory, '.wizard_state.json')
        with open(state_file, 'w') as f:
            json.dump({
                'phase': self.phase.value,
                'extract_id': self.extract_id,
                'source_url': self.source_url,
                'target_url': self.target_url,
                'enterprise_key': self.enterprise_key,
                'organizations_mapped': self.organizations_mapped,
                'validation_passed': self.validation_passed,
                'migration_run_id': self.migration_run_id
            }, f, indent=2)

    @classmethod
    def load(cls, directory: str) -> 'WizardState':
        """Load state from disk"""
        state_file = os.path.join(directory, '.wizard_state.json')
        if os.path.exists(state_file):
            with open(state_file, 'r') as f:
                data = json.load(f)
                return cls(
                    phase=WizardPhase(data['phase']),
                    extract_id=data.get('extract_id'),
                    source_url=data.get('source_url'),
                    target_url=data.get('target_url'),
                    enterprise_key=data.get('enterprise_key'),
                    organizations_mapped=data.get('organizations_mapped', False),
                    validation_passed=data.get('validation_passed', False),
                    migration_run_id=data.get('migration_run_id')
                )
        return cls(phase=WizardPhase.INIT)
