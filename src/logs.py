from datetime import datetime, UTC
from logging import getLogger, FileHandler, Formatter

LEVEL_MAPPING = {
    'debug': 10,
    'info': 20,
    'warning': 30,
    'error': 40,
    'critical': 50
}


def configure_logger(name, level, output_file=None, operation='default'):
    logger = getLogger(name)
    logger.setLevel(LEVEL_MAPPING.get(level.lower(), 0))
    if output_file is None:
        return logger
    handler = FileHandler(output_file)
    handler.setLevel(LEVEL_MAPPING.get(level.lower(), 0))
    formatter = Formatter(f'%(asctime)s | {operation} | %(levelname)s | %(message)s')
    handler.setFormatter(formatter)
    logger.addHandler(handler)
    return logger


LOGGER = None
DEFAULT_ATTRIBUTES = None


def configure_default_log_attributes(attributes: dict):
    global DEFAULT_ATTRIBUTES
    DEFAULT_ATTRIBUTES = attributes if isinstance(attributes, dict) else dict()
    return DEFAULT_ATTRIBUTES


def log_event(level: str, status: str, process_type: str, payload: dict, logger_name: str = 'default', ):
    import json
    assert isinstance(payload, dict)
    assert status in {'success', 'anomalous', 'failure', 're-queued'}
    global DEFAULT_ATTRIBUTES
    attributes = DEFAULT_ATTRIBUTES if DEFAULT_ATTRIBUTES is not None else dict()
    level_mapping = dict(
        critical=50,
        error=40,
        warning=30,
        info=20,
        debug=10
    )
    log_message = {
        "process_type": process_type,
        "created_ts": datetime.now(UTC).isoformat(timespec='seconds'),
        "status": status,
        "payload": payload,
        **attributes
    }
    getLogger(logger_name).log(level=level_mapping.get(level.lower(), 0), msg=json.dumps(log_message))
    return log_message
