"""Display helpers and user prompts for the wizard interface"""
import sys
from urllib.parse import urlparse

import click
from wizard.state import WizardPhase


PHASE_DISPLAY_NAMES = {
    WizardPhase.INIT: "Start",
    WizardPhase.EXTRACT: "Extract",
    WizardPhase.STRUCTURE: "Structure",
    WizardPhase.ORG_MAPPING: "Org Mapping",
    WizardPhase.MAPPINGS: "Mappings",
    WizardPhase.VALIDATE: "Validate",
    WizardPhase.MIGRATE: "Migrate",
    WizardPhase.PIPELINES: "Pipelines",
    WizardPhase.COMPLETE: "Complete",
}

PHASE_ORDER = [
    WizardPhase.EXTRACT,
    WizardPhase.STRUCTURE,
    WizardPhase.ORG_MAPPING,
    WizardPhase.MAPPINGS,
    WizardPhase.VALIDATE,
    WizardPhase.MIGRATE,
    WizardPhase.PIPELINES,
]


def display_welcome():
    """Display welcome message with ASCII art header"""
    click.echo()
    click.echo("=" * 60)
    click.echo("  SonarQube Server to SonarQube Cloud Migration Wizard")
    click.echo("=" * 60)
    click.echo()
    click.echo("This wizard will guide you through the migration process:")
    click.echo("  1. Extract data from SonarQube Server")
    click.echo("  2. Analyze organization structure")
    click.echo("  3. Map organizations to SonarQube Cloud")
    click.echo("  4. Generate entity mappings")
    click.echo("  5. Validate migration prerequisites")
    click.echo("  6. Execute migration")
    click.echo("  7. Update CI/CD pipelines (optional)")
    click.echo()


def display_phase_progress(current_phase: WizardPhase):
    """Display text-based progress indicator"""
    if current_phase == WizardPhase.INIT:
        current_idx = 0
    elif current_phase == WizardPhase.COMPLETE:
        current_idx = len(PHASE_ORDER)
    else:
        current_idx = PHASE_ORDER.index(current_phase) + 1

    total = len(PHASE_ORDER)
    phase_name = PHASE_DISPLAY_NAMES.get(current_phase, current_phase.value)

    click.echo()
    click.echo("-" * 60)
    click.echo(f"  [{current_idx}/{total}] {phase_name}")
    click.echo("-" * 60)


def display_phase_start(phase: WizardPhase):
    """Display phase starting message"""
    phase_name = PHASE_DISPLAY_NAMES.get(phase, phase.value)
    click.echo()
    click.echo(f">>> Starting: {phase_name}")
    click.echo()


def display_phase_complete(phase: WizardPhase):
    """Display phase completion message"""
    phase_name = PHASE_DISPLAY_NAMES.get(phase, phase.value)
    click.echo()
    click.echo(f"<<< Completed: {phase_name}")


def _read_masked_input(prompt: str) -> str:
    """Read input from the terminal, displaying asterisks for each character typed.

    Falls back to click.prompt with hidden input when not running in a real terminal
    (e.g. during testing or piped input).
    """
    try:
        import tty
        import termios
        fd = sys.stdin.fileno()
    except (AttributeError, ValueError, ImportError):
        # Not a real terminal; fall back to hidden input
        return click.prompt(prompt.rstrip(': '), hide_input=True)

    click.echo(prompt, nl=False)
    old_settings = termios.tcgetattr(fd)
    chars = []
    try:
        tty.setraw(fd)
        while True:
            ch = sys.stdin.read(1)
            if ch in ('\r', '\n'):
                break
            elif ch == '\x7f' or ch == '\x08':  # Backspace / Delete
                if chars:
                    chars.pop()
                    # Move cursor back, overwrite the asterisk, move back again
                    sys.stdout.write('\b \b')
                    sys.stdout.flush()
            elif ch == '\x03':  # Ctrl+C
                click.echo()
                raise KeyboardInterrupt
            elif ch == '\x04':  # Ctrl+D
                click.echo()
                raise EOFError
            else:
                chars.append(ch)
                sys.stdout.write('*')
                sys.stdout.flush()
    finally:
        termios.tcsetattr(fd, termios.TCSADRAIN, old_settings)
    click.echo()
    return ''.join(chars)


def prompt_credentials(prompt_text: str, hide_input: bool = True) -> str:
    """Collect credentials from user with masked input (asterisks)"""
    if hide_input:
        return _read_masked_input(f"{prompt_text}: ")
    return click.prompt(prompt_text, hide_input=False)


def _validate_server_url(url: str) -> str | None:
    """Return an error message if url is invalid, else None."""
    try:
        parsed = urlparse(url)
    except Exception:
        return "Invalid URL format. Please enter a complete URL."

    if parsed.scheme not in ("http", "https"):
        return "URL must start with http:// or https://"

    if not parsed.netloc:
        return "URL must include a valid hostname."

    return None


_LOOPBACK_HOSTNAMES = {"localhost", "127.0.0.1", "::1", "0.0.0.0"}


def _is_localhost_url(url: str) -> bool:
    """Return True if the URL targets a loopback/localhost address."""
    try:
        hostname = (urlparse(url).hostname or "").lower()
    except Exception:
        return False
    return hostname in _LOOPBACK_HOSTNAMES


def _display_localhost_docker_notice():
    """Explain Docker networking when a loopback URL is entered."""
    click.echo()
    display_warning(
        "This tool runs inside a Docker container. Inside the container, "
        "\"localhost\" refers to the container itself, not your host machine."
    )
    click.echo()
    click.echo("To connect to a SonarQube instance running on your host machine,")
    click.echo("use host.docker.internal as the hostname instead of localhost.")
    click.echo()
    click.echo("  Mac / Windows (Docker Desktop) — no extra flags needed:")
    click.echo("    Use: http://host.docker.internal:<port>")
    click.echo()
    click.echo("  Linux — re-run the tool with the --add-host flag:")
    click.echo(
        "    docker run -it --add-host=host.docker.internal:host-gateway \\\n"
        "      -v ./files:/app/files \\\n"
        "      ghcr.io/sonar-solutions/sonar-reports:latest wizard"
    )
    click.echo("    Then use: http://host.docker.internal:<port>")
    click.echo()
    click.echo("Press Ctrl+C to exit and re-run, or re-enter your URL now.")
    click.echo()


def prompt_url(prompt_text: str, default: str = None, validate: bool = False) -> str:
    """Collect URL from user"""
    while True:
        url = click.prompt(prompt_text, default=default)
        if validate:
            if _is_localhost_url(url):
                _display_localhost_docker_notice()
                continue
            error = _validate_server_url(url)
            if error:
                display_error(error)
                continue
        if not url.endswith('/'):
            url = f"{url}/"
        return url


def prompt_text(prompt_text: str, default: str = None) -> str:
    """Collect text input from user"""
    return click.prompt(prompt_text, default=default)


def display_separator():
    """Display a visual separator line between output blocks and prompts"""
    click.echo()
    click.echo("-" * 60)


def display_summary(title: str, stats_dict: dict):
    """Display formatted summary"""
    click.echo()
    click.echo(f"  {title}:")
    for key, value in stats_dict.items():
        click.echo(f"    - {key}: {value}")


def display_message(message: str):
    """Display a simple message"""
    click.echo(message)


def display_error(message: str):
    """Display an error message"""
    click.echo(click.style(f"Error: {message}", fg='red'))


def display_warning(message: str):
    """Display a warning message"""
    click.echo(click.style(f"Warning: {message}", fg='yellow'))


def display_success(message: str):
    """Display a success message"""
    click.echo(click.style(message, fg='green'))


def confirm_action(message: str, default: bool = False) -> bool:
    """Prompt for confirmation"""
    return click.confirm(message, default=default)


def confirm_review(title: str, details: dict) -> bool:
    """Display a summary of visible inputs and ask the user to accept or edit.

    Token/secret fields are intentionally excluded — they are hidden during
    input and cannot be visually verified. If the user chooses to edit, all
    credentials (including the token) are re-collected.

    Default is to edit/change (False) — user must explicitly choose to accept.
    Returns True if accepted, False if the user wants to edit.
    """
    display_message("")
    display_message(f"Please verify your {title}:")
    for label, value in details.items():
        display_message(f"  {label}: {value}")
    display_message("")
    return confirm_action("Accept this information and continue?", default=False)


def display_resume_info(state):
    """Display information about the resumable state"""
    click.echo()
    click.echo("Previous wizard session found:")
    click.echo(f"  - Current phase: {PHASE_DISPLAY_NAMES.get(state.phase, state.phase.value)}")
    if state.source_url:
        click.echo(f"  - Source URL: {state.source_url}")
    if state.extract_id:
        click.echo(f"  - Extract ID: {state.extract_id}")
    if state.target_url:
        click.echo(f"  - Target URL: {state.target_url}")
    if state.enterprise_key:
        click.echo(f"  - Enterprise Key: {state.enterprise_key}")
    click.echo()


def display_wizard_complete():
    """Display wizard completion message"""
    click.echo()
    click.echo("=" * 60)
    click.echo("  Migration Wizard Complete!")
    click.echo("=" * 60)
    click.echo()
    click.echo("Your SonarQube Server data has been migrated to SonarQube Cloud.")
    click.echo()
