#!/usr/bin/env python3
"""Embed public trust keys, upload a private ZIP, finalize and sign its release."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import sys
import tempfile
import urllib.error
import urllib.parse
import urllib.request
import zipfile
from configure import read_env


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('zip', type=Path)
    parser.add_argument('--notes', type=Path)
    parser.add_argument('--env-file', type=Path, default=Path('.env'))
    args = parser.parse_args()
    env = read_env(args.env_file) if args.env_file.exists() else {}
    env.update(os.environ)
    base = env.get('KEYGATE_BASE_URL', 'https://' + env.get('LICENSE_DOMAIN', 'license.accessible.org')).rstrip('/')
    key = env.get('RELEASE_PUBLISH_KEY', '')
    origin = urllib.parse.urlsplit(base)
    if not origin.hostname or origin.username or origin.query or origin.fragment or (origin.scheme != 'https' and env.get('KEYGATE_ALLOW_HTTP') != '1'):
        raise ValueError('Use an HTTPS licensing address; HTTP is available only for explicit local tests')
    if not key.startswith('kg_live_'):
        raise ValueError('RELEASE_PUBLISH_KEY is missing or invalid')
    opener = urllib.request.build_opener(NoRedirect)

    def api(method, path, body=None, authenticated=True):
        headers = {'Accept': 'application/json'}
        if authenticated:
            headers['Authorization'] = 'Bearer ' + key
        if body is not None:
            headers['Content-Type'] = 'application/json'
            body = json.dumps(body).encode()
        request = urllib.request.Request(base + path, data=body, headers=headers, method=method)
        try:
            with opener.open(request, timeout=120) as response:
                payload = json.load(response)
        except urllib.error.HTTPError as error:
            # Never print request headers, signed URLs, or provider secrets.
            raise ValueError('Licensing API returned HTTP ' + str(error.code)) from None
        if payload.get('success') is not True:
            raise ValueError('Licensing API did not confirm success')
        return payload.get('data', {})

    public = api('GET', '/api/v1/wordpress/accessible-forms-pro/config', authenticated=False)
    if public.get('base_url') != base or not public.get('public_keys') or not public.get('download_origins'):
        raise ValueError('The store has not created its update signing keys and private storage')
    configuration = {name: public[name] for name in ('base_url', 'product_slug', 'public_keys', 'download_origins')}
    for value in configuration['public_keys'].values():
        import base64
        if len(base64.b64decode(value, validate=True)) != 32:
            raise ValueError('Invalid release verification key')
    with zipfile.ZipFile(args.zip) as source:
        names = source.namelist()
        main_file = 'accessible-forms-pro/accessible-forms-pro.php'
        config_file = 'accessible-forms-pro/license-config.json'
        if main_file not in names or config_file not in names or len(names) != len(set(names)):
            raise ValueError('Input must be a packaged Accessible Forms Pro ZIP with one configuration file')
        code = source.read(main_file).decode('utf-8')
        match = re.search(r'^ \* Version:\s*(\S+)', code, re.M)
        if not match or not re.fullmatch(r'\d+\.\d+\.\d+(?:-[A-Za-z0-9.-]+)?', match[1]):
            raise ValueError('Plugin version header is invalid')
        version = match[1]
        for item in source.infolist():
            name = item.filename.rstrip('/')
            if not name.startswith('accessible-forms-pro/') or '\\' in name or any(part in ('', '.', '..') for part in name.split('/')) or (item.external_attr >> 16) & 0o170000 == 0o120000:
                raise ValueError('Input ZIP has an unsafe path')
        output_dir = Path('artifacts/configured')
        output_dir.mkdir(parents=True, exist_ok=True)
        output = output_dir / ('accessible-forms-pro-' + version + '.zip')
        # Deterministic timestamps/modes and JSON keep repeat runs byte-identical.
        with tempfile.NamedTemporaryFile(dir=output_dir, delete=False) as temporary:
            temporary_name = Path(temporary.name)
        try:
            with zipfile.ZipFile(temporary_name, 'w', compression=zipfile.ZIP_DEFLATED, compresslevel=9) as target:
                for name in sorted(names):
                    content = (json.dumps(configuration, sort_keys=True, indent=2) + '\n').encode() if name == config_file else source.read(name)
                    item = zipfile.ZipInfo(name, date_time=(1980, 1, 1, 0, 0, 0))
                    item.create_system = 3
                    item.external_attr = 0o100644 << 16
                    item.compress_type = zipfile.ZIP_DEFLATED
                    target.writestr(item, content, compresslevel=9)
            temporary_name.replace(output)
        finally:
            temporary_name.unlink(missing_ok=True)
    data = output.read_bytes()
    digest = hashlib.sha256(data).hexdigest()
    product_id = public['product_id']
    existing = None
    for page in range(1, 101):
        response = api('GET', '/api/v1/admin/releases?' + urllib.parse.urlencode({'product_id': product_id, 'limit': 100, 'offset': (page-1)*100}))
        releases = response.get('releases', [])
        existing = next((item for item in releases if item['version'] == version), None)
        if existing or len(releases) < 100:
            break
    if existing:
        release = api('GET', '/api/v1/admin/releases/' + existing['id'])
        artifact = next((item for item in release.get('artifacts', []) if item['platform'] == 'wordpress'), None)
        if release['status'] == 'published':
            if not artifact or artifact['sha256'] != digest:
                raise ValueError('This published version has different bytes. Use a new version; published releases are immutable.')
            print('Release ' + version + ' is already published. Configured ZIP: ' + str(output))
            return
        if release['status'] != 'draft':
            raise ValueError('This version is yanked; choose a new release version')
        if artifact:
            if artifact.get('sha256') == digest:
                api('POST', '/api/v1/admin/releases/' + release['id'] + '/actions/publish', {})
                print('Completed pending release ' + version)
                return
            api('DELETE', '/api/v1/admin/releases/' + release['id'] + '/artifacts/' + artifact['id'])
    else:
        notes = args.notes.read_text() if args.notes else 'Accessible Forms Pro ' + version + '.'
        release = api('POST', '/api/v1/admin/releases', {'product_id': product_id, 'version': version, 'channel': 'stable', 'name': 'Accessible Forms Pro ' + version, 'release_notes': notes})
    prefix = '/api/v1/admin/releases/' + release['id']
    created = api('POST', prefix + '/artifacts', {'platform': 'wordpress', 'filename': output.name, 'expected_size': len(data), 'content_type': 'application/zip'})
    request = urllib.request.Request(created['upload_url'], data=data, headers={'Content-Type': 'application/zip'}, method='PUT')
    try:
        with opener.open(request, timeout=120) as response:
            if response.status not in (200, 201, 204):
                raise ValueError('Private ZIP upload failed')
    except urllib.error.HTTPError as error:
        raise ValueError('Private ZIP upload returned HTTP ' + str(error.code)) from None
    artifact = api('POST', prefix + '/artifacts/' + created['artifact']['id'] + '/finalize', {'expected_sha256': digest})
    if artifact.get('sha256') != digest or not artifact.get('wordpress_metadata'):
        raise ValueError('Server did not confirm ZIP integrity and compatibility')
    published = api('POST', prefix + '/actions/publish', {})
    if published.get('status') != 'published':
        raise ValueError('Server did not confirm publication')
    # Save a public provenance receipt, never a secret signing key or bearer URL.
    receipt = {'version': version, 'sha256': digest, 'bytes': len(data), 'input_sha256': hashlib.sha256(args.zip.read_bytes()).hexdigest(), 'configuration': configuration, 'release_id': release['id']}
    output.with_suffix('.zip.receipt.json').write_text(json.dumps(receipt, indent=2, sort_keys=True) + '\n')
    print('Published signed release ' + version + '. Configured ZIP: ' + str(output))
    print('SHA-256: ' + digest)


if __name__ == '__main__':
    try:
        main()
    except (ValueError, OSError, urllib.error.URLError, zipfile.BadZipFile) as error:
        sys.exit('Publication failed: ' + str(error))
