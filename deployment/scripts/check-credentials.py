#!/usr/bin/env python3
"""Verify SMTP connectivity/authentication without sending email."""
from configure import read_env
import smtplib
import ssl
import sys
try:
    env = read_env('.env')
    port = int(env.get('SMTP_PORT', '587'))
    context = ssl.create_default_context()
    smtp = smtplib.SMTP_SSL(env['SMTP_HOST'], port, timeout=20, context=context) if port == 465 else smtplib.SMTP(env['SMTP_HOST'], port, timeout=20)
    with smtp:
        smtp.ehlo()
        if port != 465 and smtp.has_extn('starttls'):
            smtp.starttls(context=context)
            smtp.ehlo()
        elif port != 465 and env.get('SMTP_USERNAME'):
            raise ValueError('SMTP authentication requires TLS; use port 465 or a server with STARTTLS')
        if env.get('SMTP_USERNAME'):
            smtp.login(env['SMTP_USERNAME'], env.get('SMTP_PASSWORD', ''))
        code, _ = smtp.noop()
        if code != 250:
            raise ValueError('SMTP server did not confirm readiness')
    print('SMTP connection and credentials verified. No email was sent.')
except (OSError, ValueError, smtplib.SMTPException):
    sys.exit('SMTP verification failed. Check SMTP_HOST, SMTP_PORT, SMTP_USERNAME and SMTP_PASSWORD.')
