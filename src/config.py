"""Configuration file loader for sonar-reports"""
import json
import os
import tempfile
from typing import Dict, Any, Optional

_PATH_KEYS = frozenset({'export_directory'})


def _validate_config_paths(config: Dict[str, Any]) -> None:
    cwd_base = os.path.realpath(os.getcwd())
    tmp_base = os.path.realpath(tempfile.gettempdir())
    for key in _PATH_KEYS:
        if key in config and config[key]:
            resolved = os.path.realpath(str(config[key]))
            if not resolved.startswith(cwd_base + os.sep) and not resolved.startswith(tmp_base + os.sep):
                raise ValueError(f"Config path '{key}' must be within the working directory: {resolved}")
            config[key] = resolved


def load_config_file(config_path: str) -> Dict[str, Any]:
    """Load configuration from a JSON file.

    Args:
        config_path: Path to the JSON configuration file

    Returns:
        Dictionary containing configuration values

    Raises:
        FileNotFoundError: If the config file doesn't exist
        json.JSONDecodeError: If the config file is not valid JSON
        ValueError: If the config file path is outside the working directory
    """
    config_path = os.path.realpath(config_path)
    allowed_base = os.path.realpath(os.getcwd())
    if not (config_path.startswith(allowed_base + os.sep) or config_path == allowed_base):
        raise ValueError(f"Configuration file must be within the working directory: {config_path}")
    if not os.path.exists(config_path):
        raise FileNotFoundError(f"Configuration file not found: {config_path}")

    with open(config_path, 'r') as f:
        config = json.load(f)

    _validate_config_paths(config)
    return config


def merge_config_with_cli(config: Dict[str, Any], cli_args: Dict[str, Any]) -> Dict[str, Any]:
    """Merge configuration file values with CLI arguments.

    CLI arguments take precedence over config file values.

    Args:
        config: Configuration from JSON file
        cli_args: Arguments provided via CLI

    Returns:
        Merged configuration dictionary
    """
    merged = config.copy()

    # CLI args override config file values
    for key, value in cli_args.items():
        if value is not None:
            merged[key] = value

    return merged


def get_config_value(config: Dict[str, Any], key: str, default: Any = None) -> Any:
    """Get a configuration value with fallback to default.

    Args:
        config: Configuration dictionary
        key: Key to lookup
        default: Default value if key not found

    Returns:
        Configuration value or default
    """
    return config.get(key, default)


def validate_required_keys(config: Dict[str, Any], required_keys: list) -> None:
    """Validate that required configuration keys are present.

    Args:
        config: Configuration dictionary
        required_keys: List of required key names

    Raises:
        ValueError: If any required key is missing
    """
    missing_keys = [key for key in required_keys if key not in config or config[key] is None]

    if missing_keys:
        raise ValueError(f"Missing required configuration keys: {', '.join(missing_keys)}")
