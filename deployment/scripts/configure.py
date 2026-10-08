#!/usr/bin/env python3
"""Generate internal secrets without evaluating user-provided configuration."""
import json
import os
from pathlib import Path
import re
import secrets
import shlex
import sys


def read_env(path):
    values = {}
    for line in Path(path).read_text().splitlines():
        line = line.strip()
        if not line or line.startswith('#'):
            continue
        name, sep, value = line.partition('=')
        if not sep or not re.fullmatch(r'[A-Z][A-Z0-9_]*', name) or name in values:
            raise ValueError('Invalid or duplicate configuration name')
        # No shell evaluation, variable expansion, subprocess or command substitution.
        if value.startswith(("'", '"')):
            tokens = shlex.split(value, comments=True)
            if len(tokens) != 1:
                raise ValueError('Invalid quoted configuration value: ' + name)
            value = tokens[0]
        else:
            # Quotes within an unquoted password are literal characters.
            value = value.split(' #', 1)[0].strip()
        values[name] = value
    return values


def prepare(path):
    p = Path(path)
    values = read_env(p)
    required = ('LICENSE_DOMAIN', 'BOOTSTRAP_OWNER_EMAIL', 'STRIPE_SECRET_KEY', 'SMTP_HOST', 'SMTP_FROM')
    missing = [key for key in required if not values.get(key)]
    if missing:
        raise ValueError('Fill these configuration values: ' + ', '.join(missing))
    if not re.fullmatch(r'[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?', values['LICENSE_DOMAIN']) or '.' not in values['LICENSE_DOMAIN']:
        raise ValueError('LICENSE_DOMAIN must be a DNS hostname, without a URL or port')
    if not values['STRIPE_SECRET_KEY'].startswith(('sk_test_', 'sk_live_', 'rk_test_', 'rk_live_')):
        raise ValueError('STRIPE_SECRET_KEY must be a Stripe secret key')
    plans = json.loads(values.get('BOOTSTRAP_PLANS_JSON', '[]'))
    if not isinstance(plans, list) or not plans:
        raise ValueError('Set BOOTSTRAP_PLANS_JSON to your approved plans and prices')
    names = set()
    for plan in plans:
        if not isinstance(plan, dict) or not isinstance(plan.get('amount'), int) or isinstance(plan.get('amount'), bool) or plan['amount'] <= 0 or not isinstance(plan.get('sites'), int) or isinstance(plan.get('sites'), bool) or plan['sites'] < 0 or not re.fullmatch(r'[a-z]{3}', plan.get('currency', '')) or plan.get('interval') not in ('month', 'year', 'lifetime') or not re.fullmatch(r'[a-z0-9]+(?:[-_][a-z0-9]+)*', plan.get('slug', '')) or not plan.get('name') or plan['slug'] in names:
            raise ValueError('Each plan needs a unique slug, name, positive integer amount, currency, interval and site limit')
        names.add(plan['slug'])
    generated = {}
    for key in ('POSTGRES_PASSWORD', 'JWT_SECRET', 'LICENSE_SIGNING_KEY', 'RELEASE_KEY_ENCRYPTION_KEY', 'RELEASE_PUBLISH_KEY'):
        if not values.get(key):
            generated[key] = ('kg_live_' if key == 'RELEASE_PUBLISH_KEY' else '') + secrets.token_hex(32)
    text = p.read_text()
    for key, value in generated.items():
        if re.search(r'^' + key + r'=', text, re.M):
            text = re.sub(r'^' + key + r'=.*$', key + '=' + value, text, flags=re.M)
        else:
            text += '\n' + key + '=' + value + '\n'
    temporary = p.with_suffix('.preparing')
    temporary.write_text(text)
    temporary.chmod(0o600)
    temporary.replace(p)
    print('Configuration validated; internal secrets are ready. No secret values were printed.')


if __name__ == '__main__':
    try:
        prepare(sys.argv[1] if len(sys.argv) > 1 else '.env')
    except (ValueError, OSError, json.JSONDecodeError) as error:
        sys.exit(str(error))
