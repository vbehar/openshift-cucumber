# OpenShift Cucumber

**Cucumber steps definitions for the OpenShift API.**

[![GoDoc](https://godoc.org/github.com/vbehar/openshift-cucumber?status.svg)](https://godoc.org/github.com/vbehar/openshift-cucumber)
[![Download](https://api.bintray.com/packages/vbehar/openshift-cucumber/openshift-cucumber/images/download.svg)](https://bintray.com/vbehar/openshift-cucumber/openshift-cucumber/_latestVersion#files)
[![Travis](https://travis-ci.org/vbehar/openshift-cucumber.svg?branch=master)](https://travis-ci.org/vbehar/openshift-cucumber)
[![Circle CI](https://circleci.com/gh/vbehar/openshift-cucumber/tree/master.svg?style=svg)](https://circleci.com/gh/vbehar/openshift-cucumber/tree/master)
[![Go Report Card](http://goreportcard.com/badge/vbehar/openshift-cucumber)](http://goreportcard.com/report/vbehar/openshift-cucumber)

This project provides [Cucumber](https://github.com/cucumber/cucumber) [step definitions](https://github.com/cucumber/cucumber/wiki/Step-Definitions) to execute tests on an [OpenShift](http://www.openshift.org/) instance.

It allows you to write things like 

``` cucumber
Given My current project is "hello-openshift"
When I create a new application based on the template "hello-openshift" with parameters "APPLICATION_NAME=hello"
Then I should have a deploymentconfig "hello"
And I should have a service "hello"
...
Given I have a buildconfig "hello"
When I start a new build of "hello"
Then the latest build of "hello" should succeed in less than "5m"
...
Given I have a deploymentconfig "hello"
When the deploymentconfig "hello" has at least 1 deployment
Then the latest deployment of "hello" should succeed in less than "3m"
...
When I have a successful deployment of "hello"
Then I can access the application through the route "hello"
```

You can find more complete examples in the [examples](https://github.com/vbehar/openshift-cucumber/tree/master/examples) directory.

With it, you can write complete tests for your application deployment on OpenShift:

- login (or token validation)
- project creation
- credentials setup (secrets and serviceaccounts)
- templates creation
- applications creation
- build status
- deployment status
- route http check

It does not include all features available in OpenShift, but it's easy to add more step definitions ;-)

## Usage

Write [Cucumber feature](https://github.com/cucumber/cucumber/wiki/Feature-Introduction) files, and then just run `openshift-cucumber` with the path of yours files as argument.

Note that `openshift-cucumber` relies on environment variables to login:

* `OPENSHIFT_HOST`: the OpenShift server (for example: `https://localhost:8443`)
* `OPENSHIFT_USER`: the username (if you want to perform a login - in this case you also need to provide a password)
* `OPENSHIFT_PASSWD`: the password
* `OPENSHIFT_TOKEN`: the token (if you just want to validate a ServiceAccount token - in this case, you don't need the user or password)

If you want to run the provided examples againt a local instance of OpenShift:

```
$ export OPENSHIFT_HOST="https://localhost:8443"
$ export OPENSHIFT_USER="demo"
$ export OPENSHIFT_PASSWD="demo"
$ openshift-cucumber examples
```

### Output / Reporting

By default, `openshift-cucumber` print each step and its result (either success in green, or failure in red) in the standard output.

You can also configure a **reporter**:

* **JUnit**: the [JUnit](http://junit.org/) reporter can write the results in a [JUnit XML](http://windyroad.com.au/dl/Open%20Source/JUnit.xsd) formatted file, so that it can be used by [Jenkins](http://jenkins-ci.org/) to display a nice UI on top of it.

  You need to configure the `--reporter` and `--output` options:

  ```
  openshift-cucumber --reporter="junit" --output="/path/to/results.xml" /path/to/feature-files
  ```

## Install

Pre-build binaries for the main platforms (`darwin-amd64`, `linux-amd64` and `windows-amd64`) are available in [bintray](https://bintray.com/vbehar/openshift-cucumber/openshift-cucumber/_latestVersion#files):

 <https://bintray.com/vbehar/openshift-cucumber/openshift-cucumber/_latestVersion#files>

## Building from sources

`openshift-cucumber` is written in [Go](https://golang.org/). To build it, you need:

* [Go 1.4](http://golang.org/doc/install)
* [Godep](https://github.com/tools/godep)

Then, you can run `godep go install` to build the `openshift-cucumber` executable (in `$GOPATH/bin`)

To make things easier, you can use the provided `Dockerfile` to build in a Docker environment:

* Build a docker image:

  ```
  docker build -t openshiftcucumber -f Dockerfile.build .
  ```
* And build the project in a container (the default command is `gox`, which will build binaries for the main platforms)

  ```
  docker run --rm -v $PWD:/go/src/github.com/vbehar/openshift-cucumber openshiftcucumber
  ```
* The binaries will be available in the `build` directory.

## License

Copyright 2015 the original author or authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.