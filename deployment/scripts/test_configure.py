"""Configuration regressions: preserve secrets and fail closed before billing."""
import contextlib
import io
import json
from pathlib import Path
import tempfile
import unittest
from configure import prepare, read_env


class ConfigurationTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.path = Path(self.directory.name) / '.env'
        self.plans = [{'slug': 'test', 'name': 'Test', 'sites': 1, 'amount': 100, 'currency': 'usd', 'interval': 'year'}]
        self.configuration = 'LICENSE_DOMAIN=license.example.test\nBOOTSTRAP_OWNER_EMAIL=owner@example.test\nSTRIPE_SECRET_KEY=sk_test_fixture\nSMTP_HOST=mail.example.test\nSMTP_FROM=store@example.test\n'
        self.write()

    def write(self):
        self.path.write_text(self.configuration + "BOOTSTRAP_PLANS_JSON='" + json.dumps(self.plans) + "'\n")

    def test_idempotent_generated_secrets_and_permissions(self):
        with contextlib.redirect_stdout(io.StringIO()) as output:
            prepare(self.path)
            first = self.path.read_text()
            prepare(self.path)
        self.assertEqual(first, self.path.read_text())
        self.assertEqual(self.path.stat().st_mode & 0o777, 0o600)
        env = read_env(self.path)
        for key in ('JWT_SECRET', 'LICENSE_SIGNING_KEY', 'RELEASE_KEY_ENCRYPTION_KEY'):
            self.assertEqual(len(env[key]), 64)
            self.assertNotIn(env[key], output.getvalue())
        self.assertTrue(env['RELEASE_PUBLISH_KEY'].startswith('kg_live_'))

    def test_rejects_missing_prices_without_writing_secrets(self):
        self.plans = []
        self.write()
        before = self.path.read_text()
        with self.assertRaisesRegex(ValueError, 'approved plans'):
            prepare(self.path)
        self.assertEqual(before, self.path.read_text())

    def test_bool_is_not_a_site_limit_or_amount(self):
        for name in ('sites', 'amount'):
            self.plans[0][name] = True
            self.write()
            with self.assertRaises(ValueError):
                prepare(self.path)
            self.plans[0][name] = 1

    def test_configuration_is_not_executed_or_expanded(self):
        self.path.write_text("SMTP_PASSWORD=literal'quote$(printf secret)\nSMTP_USERNAME='name with spaces'\nJWT_SECRET=${PRIVATE_ENV}\n")
        values = read_env(self.path)
        self.assertEqual(values['SMTP_PASSWORD'], "literal'quote$(printf secret)")
        self.assertEqual(values['SMTP_USERNAME'], 'name with spaces')
        self.assertEqual(values['JWT_SECRET'], '${PRIVATE_ENV}')

    def test_rejects_duplicate_names(self):
        self.path.write_text('SMTP_HOST=first\nSMTP_HOST=second\n')
        with self.assertRaisesRegex(ValueError, 'duplicate'):
            read_env(self.path)

    def test_service_requires_hostname_and_secret_key(self):
        self.configuration = self.configuration.replace('license.example.test', 'https://license.example.test')
        self.write()
        with self.assertRaisesRegex(ValueError, 'hostname'):
            prepare(self.path)


if __name__ == '__main__':
    unittest.main()
