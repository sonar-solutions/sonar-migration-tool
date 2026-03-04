from report.utils import generate_section


def process_languages(measures, profile_map):
    languages = dict()
    for server_id, projects in measures.items():
        for project in projects.values():
            for lang_usage in project.get('ncloc_language_distribution', '').split(';'):
                try:
                    language, usage = lang_usage.split('=')
                except ValueError:
                    continue
                if language not in languages.keys():
                    languages[language] = dict(
                        language=language,
                        projects=0,
                        loc=0,
                        custom_profiles=0,
                        profiles=0,
                        base_rules=0,
                        minimum_rules=0,
                        maximum_rules=0,
                        average_rules=0,
                        total_rules=0
                    )
                languages[language]['projects'] += 1
                languages[language]['loc'] += int(usage)
    for server_id, server_languages in profile_map.items():
        for language, profiles in server_languages.items():
            if language not in languages.keys():
                continue
            for profile_key, profile in profiles.items():
                if profile_key.lower() == 'sonar way':
                    languages[language]['base_rules'] = len(profile['rules'])
                if len(profile['projects']) == 0:
                    continue
                languages[language]['profiles'] += 1
                if not profile['is_built_in']:
                    languages[language]['custom_profiles'] += 1
                languages[language]['minimum_rules'] = len(profile['rules']) if languages[language][
                                                                                    'profiles'] == 1 else min(
                    languages[language]['minimum_rules'], len(profile['rules']))
                languages[language]['maximum_rules'] = max(languages[language]['maximum_rules'], len(profile['rules']))
                languages[language]['total_rules'] += len(profile['rules'])
                languages[language]['average_rules'] = int(languages[language]['total_rules'] / languages[language][
                    'profiles'])
    return languages


def generate_language_markdown(measures, profile_map):
    languages = process_languages(measures=measures, profile_map=profile_map)
    return generate_section(
        title='Languages',
        level=3,
        headers_mapping={
            "Language": "language",
            "Lines of Code": "loc",
            "# of Projects": "projects",
            "Active Custom Profiles": "custom_profiles",
            "Sonar Way Rule Count": "base_rules",
            "Minimum Active Rule Count": "minimum_rules",
            "Maximum Active Rule Count": "maximum_rules",
            "Average Active Rule Count": "average_rules",
        },
        rows=languages.values()
    ), languages
