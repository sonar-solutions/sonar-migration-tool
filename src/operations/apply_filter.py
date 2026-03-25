from logs import log_event


def process_chunk(chunk):
    results = []
    for obj in chunk:
        if execute(**obj['kwargs']):
            results.append([True])
        else:
            results.append([])
    return results


def execute(left, right, operator, warn_message=None, **kwargs):
    allowed = True
    if operator == 'neq':
        allowed = left != right
    elif operator == 'eq':
        allowed = left == right
    elif operator == 'nin':
        allowed = left not in right
    elif operator == 'in':
        allowed = left in right
    elif operator == 'gt':
        allowed = left > right
    elif operator == 'lt':
        allowed = left < right
    elif operator == 'gte':
        allowed = left >= right
    elif operator == 'lte':
        allowed = left <= right
    if not allowed and warn_message:
        log_event(level='WARNING', status='anomalous', process_type='apply_filter',
                  payload={"message": f"{warn_message}: '{left}'"})
    return allowed
