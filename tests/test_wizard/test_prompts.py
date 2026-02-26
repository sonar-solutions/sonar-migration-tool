"""Tests for prompt and display functions in the wizard"""

from unittest.mock import patch, MagicMock
from click.testing import CliRunner
import click

from wizard.state import WizardPhase, WizardState
from wizard.prompts import (
    display_welcome,
    display_phase_progress,
    display_phase_start,
    display_phase_complete,
    display_summary,
    display_message,
    display_error,
    display_warning,
    display_success,
    display_resume_info,
    display_wizard_complete,
    prompt_credentials,
    prompt_url,
    prompt_text,
    confirm_action,
    PHASE_DISPLAY_NAMES,
    PHASE_ORDER,
    _validate_server_url,
)
from tests.test_wizard.conftest import wizard_state_at_phase


class TestDisplayWelcome:
    """Tests for display_welcome function"""

    def test_welcome_contains_sonarqube(self, capsys):
        """Welcome message should mention SonarQube"""
        display_welcome()
        captured = capsys.readouterr()
        assert 'SonarQube' in captured.out

    def test_welcome_contains_migration(self, capsys):
        """Welcome message should mention migration"""
        display_welcome()
        captured = capsys.readouterr()
        assert 'migration' in captured.out.lower() or 'Migration' in captured.out

    def test_welcome_contains_steps(self, capsys):
        """Welcome message should list the steps"""
        display_welcome()
        captured = capsys.readouterr()
        # Should contain numbered steps
        assert '1.' in captured.out
        assert '2.' in captured.out

    def test_welcome_contains_extract(self, capsys):
        """Welcome message should mention extract step"""
        display_welcome()
        captured = capsys.readouterr()
        assert 'Extract' in captured.out or 'extract' in captured.out

    def test_welcome_has_visual_separator(self, capsys):
        """Welcome message should have visual separators"""
        display_welcome()
        captured = capsys.readouterr()
        assert '=' in captured.out  # Header separator


class TestDisplayPhaseProgress:
    """Tests for display_phase_progress function"""

    def test_progress_shows_phase_name(self, capsys):
        """Progress should show the current phase name"""
        display_phase_progress(WizardPhase.EXTRACT)
        captured = capsys.readouterr()
        assert 'Extract' in captured.out

    def test_progress_shows_step_number(self, capsys):
        """Progress should show step number"""
        display_phase_progress(WizardPhase.EXTRACT)
        captured = capsys.readouterr()
        assert '[1/' in captured.out  # First phase

    def test_progress_for_each_phase(self, capsys):
        """Progress should work for all phases"""
        for i, phase in enumerate(PHASE_ORDER):
            display_phase_progress(phase)
            captured = capsys.readouterr()
            expected_num = f'[{i + 1}/'
            assert expected_num in captured.out, f"Expected {expected_num} for phase {phase}"

    def test_progress_init_phase(self, capsys):
        """Progress for INIT phase"""
        display_phase_progress(WizardPhase.INIT)
        captured = capsys.readouterr()
        assert '[0/' in captured.out

    def test_progress_complete_phase(self, capsys):
        """Progress for COMPLETE phase"""
        display_phase_progress(WizardPhase.COMPLETE)
        captured = capsys.readouterr()
        total = len(PHASE_ORDER)
        assert f'[{total}/' in captured.out


class TestDisplayPhaseStartComplete:
    """Tests for display_phase_start and display_phase_complete"""

    def test_phase_start_format(self, capsys):
        """Phase start should have >>> prefix"""
        display_phase_start(WizardPhase.STRUCTURE)
        captured = capsys.readouterr()
        assert '>>>' in captured.out
        assert 'Starting' in captured.out
        assert 'Structure' in captured.out

    def test_phase_complete_format(self, capsys):
        """Phase complete should have <<< prefix"""
        display_phase_complete(WizardPhase.STRUCTURE)
        captured = capsys.readouterr()
        assert '<<<' in captured.out
        assert 'Completed' in captured.out
        assert 'Structure' in captured.out

    def test_all_phases_have_display_names(self):
        """All phases should have display names"""
        for phase in WizardPhase:
            assert phase in PHASE_DISPLAY_NAMES, f"Missing display name for {phase}"


class TestDisplaySummary:
    """Tests for display_summary function"""

    def test_summary_shows_title(self, capsys):
        """Summary should show the title"""
        display_summary("Test Title", {"key": "value"})
        captured = capsys.readouterr()
        assert 'Test Title' in captured.out

    def test_summary_shows_key_value_pairs(self, capsys):
        """Summary should show key-value pairs"""
        display_summary("Stats", {"Count": 42, "Status": "OK"})
        captured = capsys.readouterr()
        assert 'Count' in captured.out
        assert '42' in captured.out
        assert 'Status' in captured.out
        assert 'OK' in captured.out

    def test_summary_empty_dict(self, capsys):
        """Summary should handle empty dict"""
        display_summary("Empty", {})
        captured = capsys.readouterr()
        assert 'Empty' in captured.out

    def test_summary_formatting(self, capsys):
        """Summary should have proper formatting"""
        display_summary("Test", {"A": 1})
        captured = capsys.readouterr()
        assert '-' in captured.out  # Bullet point


class TestDisplayMessages:
    """Tests for display_message, display_error, display_warning, display_success"""

    def test_display_message(self, capsys):
        """display_message should echo the message"""
        display_message("Hello World")
        captured = capsys.readouterr()
        assert 'Hello World' in captured.out

    def test_display_error(self, capsys):
        """display_error should prefix with Error"""
        display_error("Something went wrong")
        captured = capsys.readouterr()
        assert 'Error' in captured.out
        assert 'Something went wrong' in captured.out

    def test_display_warning(self, capsys):
        """display_warning should prefix with Warning"""
        display_warning("Be careful")
        captured = capsys.readouterr()
        assert 'Warning' in captured.out
        assert 'Be careful' in captured.out

    def test_display_success(self, capsys):
        """display_success should show the message"""
        display_success("Operation completed")
        captured = capsys.readouterr()
        assert 'Operation completed' in captured.out


class TestDisplayResumeInfo:
    """Tests for display_resume_info function"""

    def test_resume_info_shows_phase(self, capsys):
        """Resume info should show current phase"""
        state = wizard_state_at_phase(WizardPhase.STRUCTURE)
        display_resume_info(state)
        captured = capsys.readouterr()
        assert 'Structure' in captured.out

    def test_resume_info_shows_source_url(self, capsys):
        """Resume info should show source URL if present"""
        state = wizard_state_at_phase(
            WizardPhase.STRUCTURE,
            source_url="https://sonar.example.com/"
        )
        display_resume_info(state)
        captured = capsys.readouterr()
        assert 'https://sonar.example.com/' in captured.out

    def test_resume_info_shows_extract_id(self, capsys):
        """Resume info should show extract ID if present"""
        state = wizard_state_at_phase(
            WizardPhase.STRUCTURE,
            extract_id="1234567890"
        )
        display_resume_info(state)
        captured = capsys.readouterr()
        assert '1234567890' in captured.out

    def test_resume_info_shows_target_url(self, capsys):
        """Resume info should show target URL if present"""
        state = wizard_state_at_phase(
            WizardPhase.MIGRATE,
            target_url="https://sonarcloud.io/"
        )
        display_resume_info(state)
        captured = capsys.readouterr()
        assert 'https://sonarcloud.io/' in captured.out

    def test_resume_info_shows_enterprise_key(self, capsys):
        """Resume info should show enterprise key if present"""
        state = wizard_state_at_phase(
            WizardPhase.MIGRATE,
            enterprise_key="my-enterprise-key"
        )
        display_resume_info(state)
        captured = capsys.readouterr()
        assert 'my-enterprise-key' in captured.out

    def test_resume_info_hides_missing_values(self, capsys):
        """Resume info should not show None values"""
        state = WizardState(phase=WizardPhase.EXTRACT)
        display_resume_info(state)
        captured = capsys.readouterr()
        # Should not contain None
        assert 'None' not in captured.out


class TestDisplayWizardComplete:
    """Tests for display_wizard_complete function"""

    def test_complete_shows_success_message(self, capsys):
        """Completion should show success message"""
        display_wizard_complete()
        captured = capsys.readouterr()
        assert 'Complete' in captured.out

    def test_complete_mentions_migration(self, capsys):
        """Completion should mention migration"""
        display_wizard_complete()
        captured = capsys.readouterr()
        assert 'migrated' in captured.out.lower() or 'migration' in captured.out.lower()

    def test_complete_has_visual_separator(self, capsys):
        """Completion should have visual separators"""
        display_wizard_complete()
        captured = capsys.readouterr()
        assert '=' in captured.out


class TestPromptUrl:
    """Tests for prompt_url function"""

    def test_url_normalization_adds_trailing_slash(self):
        """URLs should have trailing slash added"""
        runner = CliRunner()
        result = runner.invoke(
            click.command()(lambda: click.echo(prompt_url("URL"))),
            input='https://sonar.example.com\n'
        )
        assert 'https://sonar.example.com/' in result.output

    def test_url_normalization_preserves_existing_slash(self):
        """URLs with trailing slash should be preserved"""
        runner = CliRunner()
        result = runner.invoke(
            click.command()(lambda: click.echo(prompt_url("URL"))),
            input='https://sonar.example.com/\n'
        )
        assert 'https://sonar.example.com/' in result.output
        assert 'https://sonar.example.com//' not in result.output

    def test_url_with_default(self):
        """URL prompt should support defaults"""
        runner = CliRunner()

        @click.command()
        def cmd():
            click.echo(prompt_url("URL", default="https://default.com/"))

        result = runner.invoke(cmd, input='\n')  # Accept default
        assert 'https://default.com/' in result.output

    def test_validate_true_accepts_valid_url(self):
        """validate=True should accept a well-formed https URL"""
        runner = CliRunner()

        @click.command()
        def cmd():
            click.echo(prompt_url("URL", validate=True))

        result = runner.invoke(cmd, input='https://sonarqube.example.com\n')
        assert 'https://sonarqube.example.com/' in result.output
        assert 'Error' not in result.output

    def test_validate_true_rejects_no_scheme_then_accepts_retry(self):
        """validate=True should reject a bare hostname and re-prompt"""
        runner = CliRunner()

        @click.command()
        def cmd():
            click.echo(prompt_url("URL", validate=True))

        # First input has no scheme; second is valid
        result = runner.invoke(cmd, input='sonarqube.example.com\nhttps://sonarqube.example.com\n')
        assert 'Error' in result.output
        assert 'http' in result.output  # error mentions http/https
        assert 'https://sonarqube.example.com/' in result.output

    def test_validate_true_rejects_localhost(self):
        """validate=True should reject http://localhost/"""
        runner = CliRunner()

        @click.command()
        def cmd():
            click.echo(prompt_url("URL", validate=True))

        result = runner.invoke(cmd, input='http://localhost/\nhttps://sonarqube.example.com\n')
        assert 'Error' in result.output
        assert 'localhost' in result.output

    def test_validate_true_rejects_127_0_0_1(self):
        """validate=True should reject http://127.0.0.1/"""
        runner = CliRunner()

        @click.command()
        def cmd():
            click.echo(prompt_url("URL", validate=True))

        result = runner.invoke(cmd, input='http://127.0.0.1/\nhttps://sonarqube.example.com\n')
        assert 'Error' in result.output

    def test_validate_false_accepts_localhost(self):
        """validate=False (default) should accept localhost without error"""
        runner = CliRunner()

        @click.command()
        def cmd():
            click.echo(prompt_url("URL"))

        result = runner.invoke(cmd, input='http://localhost:9000\n')
        assert 'Error' not in result.output
        assert 'http://localhost:9000/' in result.output


class TestValidateServerUrl:
    """Tests for the _validate_server_url helper"""

    def test_valid_https_url_returns_none(self):
        assert _validate_server_url("https://sonarqube.example.com") is None

    def test_valid_http_url_returns_none(self):
        assert _validate_server_url("http://sonarqube.example.com") is None

    def test_no_scheme_returns_error(self):
        error = _validate_server_url("sonarqube.example.com")
        assert error is not None
        assert "http" in error

    def test_localhost_returns_error(self):
        error = _validate_server_url("http://localhost")
        assert error is not None
        assert "localhost" in error

    def test_127_0_0_1_returns_error(self):
        error = _validate_server_url("http://127.0.0.1")
        assert error is not None

    def test_ipv6_loopback_returns_error(self):
        error = _validate_server_url("http://[::1]")
        assert error is not None

    def test_0_0_0_0_returns_error(self):
        error = _validate_server_url("http://0.0.0.0")
        assert error is not None


class TestPromptCredentials:
    """Tests for prompt_credentials function"""

    def test_credentials_prompt_hides_input_by_default(self):
        """Credentials should be hidden by default"""
        runner = CliRunner()

        @click.command()
        def cmd():
            # The hide_input parameter should be True by default
            result = prompt_credentials("Token")
            click.echo(f"Got: {result}")

        result = runner.invoke(cmd, input='secret-token\n')
        # Input should not be echoed
        assert 'secret-token' not in result.output.split('\n')[0]  # Not in prompt line

    def test_credentials_returns_correct_value(self):
        """Credentials prompt should return the entered value"""
        runner = CliRunner()

        @click.command()
        def cmd():
            result = prompt_credentials("Token")
            click.echo(f"Got: {result}")

        result = runner.invoke(cmd, input='secret-token\n')
        assert 'Got: secret-token' in result.output

    def test_credentials_can_show_input(self):
        """Credentials can optionally show input"""
        runner = CliRunner()

        @click.command()
        def cmd():
            result = prompt_credentials("Token", hide_input=False)
            click.echo(f"Got: {result}")

        result = runner.invoke(cmd, input='visible-token\n')
        assert 'Got: visible-token' in result.output

    def test_masked_input_shows_asterisks(self):
        """When running in a real terminal, asterisks should be displayed"""
        mock_stdin_chars = list('mytoken') + ['\r']
        written_chars = []

        mock_fd = 3
        mock_old_settings = [0, 0, 0, 0, 0, 0, [b'\x00'] * 32]

        with patch('wizard.prompts.sys') as mock_sys, \
             patch('wizard.prompts.click') as _mock_click:
            mock_sys.stdin.fileno.return_value = mock_fd
            mock_sys.stdin.read.side_effect = mock_stdin_chars
            mock_sys.stdout.write.side_effect = lambda c: written_chars.append(c)
            mock_sys.stdout.flush.return_value = None

            with patch('tty.setraw') as _mock_setraw, \
                 patch('termios.tcgetattr', return_value=mock_old_settings), \
                 patch('termios.tcsetattr'):
                from wizard.prompts import _read_masked_input
                result = _read_masked_input("Token: ")

        assert result == 'mytoken'
        assert written_chars == ['*'] * len('mytoken')

    def test_masked_input_handles_backspace(self):
        """Backspace should remove the last character and its asterisk"""
        # Type 'abc', backspace, 'd' -> result should be 'abd'
        mock_stdin_chars = list('abc') + ['\x7f'] + list('d') + ['\r']
        written_chars = []

        mock_fd = 3
        mock_old_settings = [0, 0, 0, 0, 0, 0, [b'\x00'] * 32]

        with patch('wizard.prompts.sys') as mock_sys, \
             patch('wizard.prompts.click') as _mock_click:
            mock_sys.stdin.fileno.return_value = mock_fd
            mock_sys.stdin.read.side_effect = mock_stdin_chars
            mock_sys.stdout.write.side_effect = lambda c: written_chars.append(c)
            mock_sys.stdout.flush.return_value = None

            with patch('tty.setraw'), \
                 patch('termios.tcgetattr', return_value=mock_old_settings), \
                 patch('termios.tcsetattr'):
                from wizard.prompts import _read_masked_input
                result = _read_masked_input("Token: ")

        assert result == 'abd'
        # Should have: *, *, *, \b \b (backspace sequence), *
        assert '\b \b' in ''.join(written_chars)


class TestPromptText:
    """Tests for prompt_text function"""

    def test_text_prompt_basic(self):
        """Basic text prompt should work"""
        runner = CliRunner()

        @click.command()
        def cmd():
            result = prompt_text("Enter name")
            click.echo(f"Name: {result}")

        result = runner.invoke(cmd, input='TestName\n')
        assert 'Name: TestName' in result.output

    def test_text_prompt_with_default(self):
        """Text prompt should support defaults"""
        runner = CliRunner()

        @click.command()
        def cmd():
            result = prompt_text("Enter name", default="DefaultName")
            click.echo(f"Name: {result}")

        result = runner.invoke(cmd, input='\n')
        assert 'Name: DefaultName' in result.output


class TestConfirmAction:
    """Tests for confirm_action function"""

    def test_confirm_default_false(self):
        """Confirm with default=False should require explicit yes"""
        runner = CliRunner()

        @click.command()
        def cmd():
            result = confirm_action("Proceed?", default=False)
            click.echo(f"Result: {result}")

        # Empty input with default=False should return False
        result = runner.invoke(cmd, input='\n')
        assert 'Result: False' in result.output

    def test_confirm_default_true(self):
        """Confirm with default=True should default to yes"""
        runner = CliRunner()

        @click.command()
        def cmd():
            result = confirm_action("Proceed?", default=True)
            click.echo(f"Result: {result}")

        # Empty input with default=True should return True
        result = runner.invoke(cmd, input='\n')
        assert 'Result: True' in result.output

    def test_confirm_explicit_yes(self):
        """Confirm should accept 'y' as yes"""
        runner = CliRunner()

        @click.command()
        def cmd():
            result = confirm_action("Proceed?", default=False)
            click.echo(f"Result: {result}")

        result = runner.invoke(cmd, input='y\n')
        assert 'Result: True' in result.output

    def test_confirm_explicit_no(self):
        """Confirm should accept 'n' as no"""
        runner = CliRunner()

        @click.command()
        def cmd():
            result = confirm_action("Proceed?", default=True)
            click.echo(f"Result: {result}")

        result = runner.invoke(cmd, input='n\n')
        assert 'Result: False' in result.output


class TestPhaseDisplayNamesCompleteness:
    """Tests for PHASE_DISPLAY_NAMES completeness"""

    def test_all_phases_have_display_names(self):
        """Every WizardPhase should have a display name"""
        for phase in WizardPhase:
            assert phase in PHASE_DISPLAY_NAMES, f"Missing display name for {phase}"
            assert isinstance(PHASE_DISPLAY_NAMES[phase], str)
            assert len(PHASE_DISPLAY_NAMES[phase]) > 0

    def test_phase_order_contains_workflow_phases(self):
        """PHASE_ORDER should contain all workflow phases"""
        workflow_phases = [
            WizardPhase.EXTRACT,
            WizardPhase.STRUCTURE,
            WizardPhase.ORG_MAPPING,
            WizardPhase.MAPPINGS,
            WizardPhase.VALIDATE,
            WizardPhase.MIGRATE,
            WizardPhase.PIPELINES,
        ]
        for phase in workflow_phases:
            assert phase in PHASE_ORDER, f"Missing phase {phase} from PHASE_ORDER"

    def test_phase_order_excludes_init_and_complete(self):
        """PHASE_ORDER should not contain INIT or COMPLETE"""
        assert WizardPhase.INIT not in PHASE_ORDER
        assert WizardPhase.COMPLETE not in PHASE_ORDER


class TestColoredOutput:
    """Tests for colored output in display functions"""

    def test_error_uses_red_color(self, capsys):
        """Error messages should be styled (color applied via click)"""
        display_error("Test error")
        captured = capsys.readouterr()
        # Can't easily test ANSI codes in capsys, but we verify the message is there
        assert 'Test error' in captured.out

    def test_warning_uses_yellow_color(self, capsys):
        """Warning messages should be styled"""
        display_warning("Test warning")
        captured = capsys.readouterr()
        assert 'Test warning' in captured.out

    def test_success_uses_green_color(self, capsys):
        """Success messages should be styled"""
        display_success("Test success")
        captured = capsys.readouterr()
        assert 'Test success' in captured.out
