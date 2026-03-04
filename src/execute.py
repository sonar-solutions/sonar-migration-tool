import os

from logs import log_event
from dependencies import load_dependencies
from utils import export_jsonl


def execute_plan(execution_plan, inputs, concurrency, task_configs, output_directory, current_run_id, run_ids):
    log_event(level='WARNING', status='success', process_type='execute_plan',
              payload=dict(message=list(run_ids)))
    for idx, phase in enumerate(execution_plan):
        log_event(level='WARNING', status='success', process_type='execute_plan', payload=dict(message=f"Executing phase {idx + 1} of {len(execution_plan)}"))

        execute_phase(phase=phase, inputs=inputs, concurrency=concurrency, task_configs=task_configs,
                      output_directory=output_directory, current_run_id=current_run_id, run_ids=run_ids)


def execute_phase(phase, inputs, concurrency, task_configs, output_directory, current_run_id, run_ids):
    for task in phase:
        log_event(level='WARNING', status='success', process_type='execute_phase', payload=dict(message=f"Executing task {task} - {phase.index(task) + 1} of {len(phase)}"))
        execute_task(task=task, task_config=task_configs[task], inputs=inputs, concurrency=concurrency,
                     output_directory=output_directory, run_id=current_run_id, run_ids=run_ids)


def execute_task(task, concurrency, inputs, task_config, output_directory, run_id, run_ids):
    dependencies = load_dependencies(
        inputs=inputs,
        task=task,
        task_config=task_config,
        concurrency=concurrency,
        output_directory=output_directory,
        run_ids=run_ids
    )
    export_dir = os.path.join(output_directory, run_id) + '/'
    os.makedirs(os.path.join(export_dir, task), exist_ok=True)
    for idx, chunk in enumerate(dependencies):
        output = True
        for op_idx, operation_config in enumerate(task_config['operations']):
            chunk = execute_operation(task=task, operation_config=operation_config, idx=idx, op_idx=op_idx, total_ops=len(task_config['operations']), chunk=chunk)
            if not chunk:
                output = False
                break
        if output:
            export_jsonl(directory=export_dir, name=task, data=chunk, idx=idx)


def execute_operation(task, operation_config, idx, op_idx, total_ops, chunk):
    from operations import load_operation
    from parser import extract_inputs
    log_event(level='WARNING', status='success', process_type='execute_task',
              payload=dict(message=f"Executing {task} operation {op_idx+1} of {total_ops}: {operation_config['operation']} on chunk {idx + 1}"))
    op = load_operation(name=operation_config['operation'])
    inputs = [extract_inputs(obj=obj, operation_config=operation_config) for obj in chunk]
    res = op.process_chunk(chunk=inputs)
    results = list()
    for idx, r in enumerate(res):
        for obj in r:
            if obj is None:
                continue
            elif isinstance(obj, bool) and obj is True:
                results.append(chunk[idx])
            elif isinstance(obj, dict):
                new_obj = {k:v for k,v in chunk[idx]['obj'].items() if k not in obj.keys()}
                new_obj.update(obj)
                results.append(dict(obj=new_obj, **{k: v for k, v in chunk[idx].items() if k != 'obj'}))
    return results
