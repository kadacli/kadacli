# Security Policy

## Reporting a vulnerability

If you believe you have found a security vulnerability in kåda, please report it
privately. **Do not open a public issue.**

Email **hello@kadacli.io** with:

- a description of the issue and its impact,
- the steps required to reproduce it, and
- any relevant logs, configuration, or proof-of-concept (with secrets redacted).

We will acknowledge your report as soon as we can and keep you informed as we
work on a fix. Please give us a reasonable amount of time to address the issue
before any public disclosure.

## Supported versions

kåda is pre-1.0 / early release. Security fixes are applied to the latest
release only until a formal support policy is published.

## A note on credentials

kåda reads cloud credentials from your environment (for example
`AZURE_SUBSCRIPTION_ID` and a service principal). **Never commit credentials,
`.env` files, or secrets to a repository.** Use your CI/CD platform's secret
store and short-lived, least-privilege identities. If a credential is ever
exposed, rotate it immediately.
