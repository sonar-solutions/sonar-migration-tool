from report.utils import generate_section


def generate_ide_markdown(users):
    row = dict(
        total=len([user for user in sum(users.values(), []) if user['sonar_lint_connection']]),
        active=len([user for user in sum(users.values(), []) if user['is_active_sonar_lint']])
    )
    return generate_section(
        title="IDE Integration",
        headers_mapping={
            "Connected Mode Users": "total",
            "Active Connected Mode Users": "active"
        },
        rows=[row]
    )
