# auth-z

auth-z is an identity management tool utilizing OpenID Connect for authenticating users across various applications.

auth-z functions as an intermediary to other identity systems via "connectors," allowing it to delegate authentication to services like LDAP, SAML, or popular providers like GitHub, Google, and Active Directory. Developers can interact with auth-z once, while auth-z handles the complex protocols for each backend.

## ID Tokens

ID Tokens, a key component of OAuth2 extended by OpenID Connect, are auth-z's main functionality. These tokens are JSON Web Tokens (JWTs), signed by auth-z, that authenticate the user within an OAuth2 response. An example JWT might resemble:

```
eyJhbGciOiJSUzI1NiIsImtpZCI6IjlkNDQ3NDFmNzczYjkzOGNmNjVkZDMyNjY4NWI4NjE4MGMzMjRkOTkifQ.eyJpc3MiOiJodHRwOi8vMTI3LjAuMC4xOjU1NTYvZGV4Iiwic3ViIjoiQ2djeU16UXlOelE1RWdabmFYUm9kV0kiLCJhdWQiOiJleGFtcGxlLWFwcCIsImV4cCI6MTQ5Mjg4MjA0MiwiaWF0IjoxNDkyNzk1NjQyLCJhdF9oYXNoIjoiYmk5NmdPWFpTaHZsV1l0YWw5RXFpdyIsImVtYWlsIjoiZXJpYy5jaGlhbmdAY29yZW9zLmNvbSIsImVtYWlsX3ZlcmlmaWVkIjp0cnVlLCJncm91cHMiOlsiYWRtaW5zIiwiZGV2ZWxvcGVycyJdLCJuYW1lIjoiRXJpYyBDaGlhbmcifQ.OhROPq_0eP-zsQRjg87KZ4wGkjiQGnTi5QuG877AdJDb3R2ZCOk2Vkf5SdP8cPyb3VMqL32G4hLDayniiv8f1_ZXAde0sKrayfQ10XAXFgZl_P1yilkLdknxn6nbhDRVllpWcB12ki9vmAxklAr0B1C4kr5nI3-BZLrFcUR5sQbxwJj4oW1OuG6jJCNGHXGNTBTNEaM28eD-9nhfBeuBTzzO7BKwPsojjj4C9ogU4JQhGvm_l4yfVi0boSx8c0FX3JsiB0yLa1ZdJVWVl9m90XmbWRSD85pNDQHcWZP9hR6CMgbvGkZsgjG32qeRwUL_eNkNowSBNWLrGNPoON1gMg
```

ID Tokens include standard claims specifying which app authenticated the user, when the token expires, and the user’s identity.

```json
{
  "iss": "http://127.0.0.1:5556/dex",
  "sub": "CgcyMzQyNzQ5EgZnaXRodWI",
  "aud": "example-app",
  "exp": 1492882042,
  "iat": 1492795642,
  "at_hash": "bi96gOXZShvlWYtal9Eqiw",
  "email": "jane.doe@coreos.com",
  "email_verified": true,
  "groups": [
    "admins",
    "developers"
  ],
  "name": "Jane Doe"
}
```

Signed by auth-z and containing standard claims, these tokens can serve as credentials between services. Systems supporting auth-z’s OpenID Connect tokens include Kubernetes and AWS STS.

For instructions on creating or verifying an ID Token, refer to the section "Writing apps that use auth-z."

## Kubernetes Integration

auth-z seamlessly integrates with Kubernetes clusters, supporting Custom Resource Definitions and enabling API authentication through OpenID Connect. Clients, such as the `kubernetes-dashboard` and `kubectl`, can use auth-z to authenticate users accessing the cluster.

* Documentation for setting up auth-z on Kubernetes is available.
* View companies and projects utilizing auth-z in ADOPTERS.md.

## Connectors

auth-z serves as a bridge to external user management systems like LDAP, GitHub, and more. Acting as a connector, auth-z enables clients to use OpenID Connect for communication, while auth-z manages authentication with the identity provider.

A "connector" allows auth-z to authenticate users against a specific identity provider, supporting platforms such as GitHub, LinkedIn, and Microsoft, as well as protocols like LDAP and SAML.

Certain connectors have limitations; for example, SAML doesn’t support refresh tokens, so users authenticated via SAML won’t receive refresh tokens from auth-z, which may impact offline access for applications like `kubectl`.

auth-z supports these connectors:

| Name               | Refresh Tokens Supported | Groups Claim Supported | Preferred Username Claim Supported | Status | Notes                                                            |
| ------------------ | ------------------------ | ---------------------- | ---------------------------------- | ------ | ---------------------------------------------------------------- |
| LDAP               | yes                      | yes                    | yes                                | stable |                                                                  |
| GitHub             | yes                      | yes                    | yes                                | stable |                                                                  |
| SAML 2.0           | no                       | yes                    | no                                 | stable | WARNING: SAML is vulnerable to auth bypasses                     |
| GitLab             | yes                      | yes                    | yes                                | beta   |                                                                  |
| OpenID Connect     | yes                      | yes                    | yes                                | beta   | Covers Salesforce, Azure, etc.                                   |
| OAuth 2.0          | no                       | yes                    | yes                                | alpha  |                                                                  |
| Google             | yes                      | yes                    | yes                                | alpha  |                                                                  |
| LinkedIn           | yes                      | no                     | no                                 | beta   |                                                                  |
| Microsoft          | yes                      | yes                    | no                                 | beta   |                                                                  |
| AuthProxy          | no                       | yes                    | no                                 | alpha  | Suitable for authentication proxies like Apache mod_auth         |
| Bitbucket Cloud    | yes                      | yes                    | no                                 | alpha  |                                                                  |
| OpenShift          | yes                      | yes                    | no                                 | alpha  |                                                                  |
| Atlassian Crowd    | yes                      | yes                    | yes *                              | beta   | * Requires preferred_username claim setup                        |
| Gitea              | yes                      | no                     | yes                                | beta   |                                                                  |
| OpenStack Keystone | yes                      | yes                    | no                                 | alpha  |                                                                  |

Connector statuses:
* Stable: Highly tested and widely used, with stable API.
* Beta: Tested, minor backward compatibility changes possible.
* Alpha: Experimental and subject to change.

Changes to connectors will be outlined in the release notes.

## Documentation

* Getting started
* OpenID Connect Overview
* Using auth-z in Applications
* Version 2 Updates
* Custom claims and client options
* Data storage options
* gRPC API
* auth-z and Kubernetes
* Client SDKs

## Reporting Issues

To report a vulnerability, see our security policy.

## Support

- Submit feature requests and bug reports through issues.
- Join the conversation:
  - #auth-z on CNCF Slack
  - Start a discussion
  - Subscribe to the auth-z-dev mailing list

## Developer Setup

After coding and testing, execute all tests:

```shell
make testall
```

For a streamlined setup, install Nix and direnv.

Alternatively, install Go and Docker manually or via a package manager, then run `make deps` for other dependencies.

See release documentation for guidance on the release process.

## License

This project is licensed under the Apache License, Version 2.0.
