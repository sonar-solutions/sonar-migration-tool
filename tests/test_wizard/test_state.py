import json
import os
import tempfile

from wizard.state import WizardPhase, WizardState


class TestWizardPhase:
    def test_wizard_phase_values(self):
        """Verify enum values match expected strings"""
        assert WizardPhase.INIT.value == "init"
        assert WizardPhase.EXTRACT.value == "extract"
        assert WizardPhase.STRUCTURE.value == "structure"
        assert WizardPhase.ORG_MAPPING.value == "org_mapping"
        assert WizardPhase.MAPPINGS.value == "mappings"
        assert WizardPhase.VALIDATE.value == "validate"
        assert WizardPhase.MIGRATE.value == "migrate"
        assert WizardPhase.PIPELINES.value == "pipelines"
        assert WizardPhase.COMPLETE.value == "complete"

    def test_wizard_phase_from_string(self):
        """Verify enum can be created from string values"""
        assert WizardPhase("init") == WizardPhase.INIT
        assert WizardPhase("extract") == WizardPhase.EXTRACT
        assert WizardPhase("complete") == WizardPhase.COMPLETE


class TestWizardState:
    def test_wizard_state_defaults(self):
        """Verify default state is INIT with None fields"""
        state = WizardState(phase=WizardPhase.INIT)
        assert state.phase == WizardPhase.INIT
        assert state.extract_id is None
        assert state.source_url is None
        assert state.target_url is None
        assert state.enterprise_key is None
        assert state.organizations_mapped is False
        assert state.validation_passed is False
        assert state.migration_run_id is None

    def test_wizard_state_with_values(self):
        """Verify state can be created with all values"""
        state = WizardState(
            phase=WizardPhase.MIGRATE,
            extract_id="extract-123",
            source_url="https://source.example.com",
            target_url="https://target.example.com",
            enterprise_key="enterprise-key",
            organizations_mapped=True,
            validation_passed=True,
            migration_run_id="run-456"
        )
        assert state.phase == WizardPhase.MIGRATE
        assert state.extract_id == "extract-123"
        assert state.source_url == "https://source.example.com"
        assert state.target_url == "https://target.example.com"
        assert state.enterprise_key == "enterprise-key"
        assert state.organizations_mapped is True
        assert state.validation_passed is True
        assert state.migration_run_id == "run-456"

    def test_save_and_load_roundtrip(self):
        """Save state, load it, verify equality"""
        with tempfile.TemporaryDirectory() as tmpdir:
            original = WizardState(
                phase=WizardPhase.EXTRACT,
                extract_id="test-extract-id",
                source_url="https://source.test",
                target_url="https://target.test",
                enterprise_key="test-key",
                organizations_mapped=True,
                validation_passed=False,
                migration_run_id=None
            )
            original.save(tmpdir)
            loaded = WizardState.load(tmpdir)

            assert loaded.phase == original.phase
            assert loaded.extract_id == original.extract_id
            assert loaded.source_url == original.source_url
            assert loaded.target_url == original.target_url
            assert loaded.enterprise_key == original.enterprise_key
            assert loaded.organizations_mapped == original.organizations_mapped
            assert loaded.validation_passed == original.validation_passed
            assert loaded.migration_run_id == original.migration_run_id

    def test_load_missing_file(self):
        """Returns INIT state when file doesn't exist"""
        with tempfile.TemporaryDirectory() as tmpdir:
            state = WizardState.load(tmpdir)
            assert state.phase == WizardPhase.INIT
            assert state.extract_id is None

    def test_load_partial_data(self):
        """Handles missing optional fields gracefully"""
        with tempfile.TemporaryDirectory() as tmpdir:
            state_file = os.path.join(tmpdir, '.wizard_state.json')
            with open(state_file, 'w') as f:
                json.dump({
                    'phase': 'extract'
                }, f)

            loaded = WizardState.load(tmpdir)
            assert loaded.phase == WizardPhase.EXTRACT
            assert loaded.extract_id is None
            assert loaded.source_url is None
            assert loaded.organizations_mapped is False
            assert loaded.validation_passed is False

    def test_save_creates_file(self):
        """Verify save creates the state file"""
        with tempfile.TemporaryDirectory() as tmpdir:
            state = WizardState(phase=WizardPhase.STRUCTURE)
            state.save(tmpdir)

            state_file = os.path.join(tmpdir, '.wizard_state.json')
            assert os.path.exists(state_file)

            with open(state_file, 'r') as f:
                data = json.load(f)
            assert data['phase'] == 'structure'

    def test_save_overwrites_existing(self):
        """Verify save overwrites existing state file"""
        with tempfile.TemporaryDirectory() as tmpdir:
            state1 = WizardState(phase=WizardPhase.INIT)
            state1.save(tmpdir)

            state2 = WizardState(phase=WizardPhase.COMPLETE, extract_id="new-id")
            state2.save(tmpdir)

            loaded = WizardState.load(tmpdir)
            assert loaded.phase == WizardPhase.COMPLETE
            assert loaded.extract_id == "new-id"
