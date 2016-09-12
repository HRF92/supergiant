SUPERGIANT: Easy container orchestration using Kubernetes
=========================================================

---

<!-- Links -->

[Kubernetes Source URL]: https://github.com/kubernetes/kubernetes
[Supergiant Website URL]: https://supergiant.io/
[Supergiant Docs URL]: https://supergiant.io/docs
[Supergiant Tutorials URL]: https://supergiant.io/tutorials
[Supergiant Slack URL]: https://supergiant.io/slack
[Supergiant Community URL]: https://supergiant.io/community
[Supergiant Contribution Guidelines URL]: http://supergiant.github.io/docs/community/contribution-guidelines.html
[Supergiant Swagger Docs URL]: http://swagger.supergiant.io/docs/
[Tutorial AWS URL]: https://supergiant.io/blog/how-to-install-supergiant-container-orchestration-engine-on-aws-ec2?utm_source=github
[Tutorial MongoDB URL]: https://supergiant.io/blog/deploy-a-mongodb-replica-set-with-docker-and-supergiant?urm_source=github
[Community and Contributing Anchor]: #community-and-contributing
[Swagger URL]: http://swagger.io/
[Git URL]: https://git-scm.com/
[Go URL]: https://golang.org/
[godep URL]: https://github.com/tools/godep
[Supergiant Go Package Anchor]: #how-to-install-supergiant-as-a-go-package
[Generate CSR Anchor]: #how-to-generate-a-certificate-signing-request-file
[Create Admin User Anchor]: #how-to-create-an-admin-user
[Install Dependencies Anchor]: #installing-generating-dependencies

<!-- Badges -->

[GoReportCard Widget]: https://goreportcard.com/badge/github.com/supergiant/supergiant
[GoReportCard URL]: https://goreportcard.com/report/github.com/supergiant/supergiant
[GoDoc Widget]: https://godoc.org/github.com/supergiant/supergiant?status.svg
[GoDoc URL]: https://godoc.org/github.com/supergiant/supergiant
[Travis Widget]: https://travis-ci.org/supergiant/supergiant.svg?branch=master
[Travis URL]: https://travis-ci.org/supergiant/supergiant
[Release Widget]: https://img.shields.io/github/release/supergiant/supergiant.svg
[Release URL]: https://github.com/supergiant/supergiant/releases/latest
[Swagger API Widget]: http://online.swagger.io/validator?url=http://swagger.supergiant.io/api-docs
[Swagger URL]: http://swagger.supergiant.io/docs/

### <img src="http://supergiant.io/img/logo_dark.svg" width="400">

[![GoReportCard Widget]][GoReportCard URL] [![GoDoc Widget]][GoDoc URL] [![Travis Widget]][Travis URL] [![Release Widget]][Release URL]

---


Supergiant is an open-source container orchestration system that lets developers easily deploy and manage apps as Docker containers using Kubernetes.

We want to make Supergiant the easiest way to run Kubernetes in the cloud.

Quick start...

* [How to Install Supergiant Container Orchestration Engine on AWS EC2][Tutorial AWS URL]
* [Deploy a MongoDB Replica Set with Docker and Supergiant][Tutorial MongoDB URL]

---

## Features

* Lets you manage microservices with Docker containers
* Lets you manage multiple users (OAUTH and LDAP coming soon)
* Web dashboard served over HTTPS/SSL by default
* Manages hardware like one, big self-healing resource pool
* Lets you easily scale stateful services and HA volumes on the fly
* Lowers costs by auto-scaling hardware when needed
* Lets you easily adjust RAM and CPU min and max values independently for each service
* Manages hardware topology organically within configurable constraints


## Resources

* [Supergiant Website][Supergiant Website URL]
* [Docs](https://supergiant.io/docs)
* [Tutorials](https://supergiant.io/tutorials)
* [Slack](https://supergiant.io/slack)
* [Install][Tutorial AWS URL]


## Installation

The current release installs on Amazon Web Services EC2, using a
publicly-available AMI. Other cloud providers and local installation are in
development.

If you want to install Supergiant, follow the [Supergiant Install Tutorial][Tutorial AWS URL].


## Top-Level Concepts

Supergiant makes it easy to run Dockerized apps as services in the cloud by
abstracting Kubernetes resources. It doesn’t obscure Kubernetes in any way --
in fact you could simply use Supergiant to install Kubernetes.

Supergiant abstracts Kubernetes and cloud provider services into a few
easily-managed resources, namely:

* **Apps** are what groups Components into Kubernetes Namespaces. An App is how
to organize some collection of (micro)services in an environment, such as
"my-app-production.”Organization is flexible and up to the user.

* **Entrypoints** allow Components to be reached through a public,
internet-facing address. They are how Supergiant handles external load
balancing. Kubernetes handles internal load balancing among containers
brilliantly, so we use Entrypoints as a more efficient system for external load balancing among Nodes.

* **A Component** is child of an App and is synonymous with microservice; in
that, a Component should ideally have one role or responsibility within an App.
As a basic example: within an App named "wordpress-production", there might be
two Components named "mysql" and "wordpress".

* **A Release** is a configuration of a Component, released at a certain time.
Releases can be verbose, as they represent an entire topology of a Component,
it’s storage volumes, its min and max allocated resources, etc. By managing
Docker instances as Releases, HA storage volumes can be attached and reattached
without losing statefulness.

Supergiant makes use of the [Swagger API framework][Swagger URL] for documenting all resources. See the full Supergiant API documentation for the full reference.

* [![Swagger API Widget] Supergiant Swagger API reference][Supergiant Swagger Docs URL]


## Micro-Roadmap

Currently, the core team is working on the following:

* Add LDAP and OAUTH user authentication
* Add support for additional cloud providers
* Add support for local installations


## Community and Contributing

We are very grateful of any contribution.

All Supergiant projects require familiarization with our Community and our Contribution Guidelines. Please see these links to get started.

* [Community Page][Supergiant Community URL]
* [Contribution Guidelines][Supergiant Contribution Guidelines URL]


## Development

If you would like to contribute changes to Supergiant, first see the pages in
the section above, [Community and Contributing][Community and Contributing Anchor].

In order to set up a development environment be sure you have the following dependencies ready:

* [Git][Git URL]
* [Go][Go URL] version 1.7 or more recent
* [godep][godep URL]
* [Supergiant as a Go package][Supergiant Go Package Anchor]
* [A certificate signing request file][Generate CSR Anchor] for `localhost`
* [A Supergiant Admin user][Create Admin User Anchor]

If you are missing any of these, see below to [install or generate dependencies][Install Dependencies Anchor].

From the supergiant Go package folder (usually ~/.go/src/github.com/supergiant/supergiant)

#### Run Supergiant

```shell
godep go run main.go --config-file config/config.json
```

Local Supergiant will expect HTTPS requests on port `8081`. Access the dashboard at https://localhost:8081/ui with the Admin username and password [you generated][Create Admin User Anchor].


#### Run Tests

```shell
godep go test -v ./test/...
```


---

## Installing/Generating Dependencies

#### How to Install Supergiant as a Go Package

```shell
go get github.com/supergiant/supergiant
```

#### How to Generate a Certificate Signing Request File

You will need local Supergiant RSA `.key` and `.pem` files. The default locations are in the Supergiant Go package tmp folder (usually `~/.go/src/github.com/supergiant/supergiant`) as `tmp/supergiant.key`, `tmp/supergiant.pem`. If you wish to customize these locations, you will need to edit `config/config.json`. The following steps require no config editing.

Set the following env session variables:

```shell
SSL_KEY_FILE=tmp/supergiant.key
SSL_CRT_FILE=tmp/supergiant.pem
```

Generate the `.key` file

```shell
openssl genrsa -out $SSL_KEY_FILE 2048
```

Generate the CSR file. This step will ask you a few questions about the computer you are using to generate the file.

When you are asked to enter **Common Name (e.g. server FQDN or YOUR name)**, enter `localhost`.

```shell
openssl req -new -x509 -sha256 -key $SSL_KEY_FILE -out $SSL_CRT_FILE -days 3650
```

_Note: You may customize this in `config/config.json`, but custom configuration and generating a custom CSR is outside the scope of this _HowTo_._

#### How to Create an Admin User

From the supergiant Go package folder (usually `~/.go/src/github.com/supergiant/supergiant`) run the
following:

```shell
godep go run cmd/generate_admin_user/generate_admin_user.go --config-file config/config.json
```

---

## License

This software is licensed under the Apache License, version 2 ("ALv2"), quoted below.

Copyright 2016 Qbox, Inc., a Delaware corporation. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License"); you may not
use this file except in compliance with the License. You may obtain a copy of
the License at http://www.apache.org/licenses/LICENSE-2.0.

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
License for the specific language governing permissions and limitations under
the License.
