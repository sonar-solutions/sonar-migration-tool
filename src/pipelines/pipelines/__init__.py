from pipelines.utils import load_module

def identify_pipeline_type(platform, file):
    pipeline_types = platform.get_available_pipelines()
    pipeline_mod = None
    for pipeline_type in pipeline_types:
        mod = load_module(mod_type='pipelines', name=pipeline_type)
        if mod is not None and mod.is_valid_pipeline(file=file):
            pipeline_mod = mod
            break
    return pipeline_mod