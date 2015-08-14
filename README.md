# OpenShift Cucumber

**Cucumber steps definitions for the OpenShift API.**

[![GoDoc](https://godoc.org/github.com/vbehar/openshift-cucumber?status.svg)](https://godoc.org/github.com/vbehar/openshift-cucumber)

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

- login
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
* `OPENSHIFT_USER`: the username
* `OPENSHIFT_PASSWD`: the password

If you want to run the provided examples againt a local instance of OpenShift:

```
$ export OPENSHIFT_HOST="https://localhost:8443"
$ export OPENSHIFT_USER="demo"
$ export OPENSHIFT_PASSWD="demo"
$ openshift-cucumber examples
```

## Building

`openshift-cucumber` is written in [Go](https://golang.org/). To build it, you need:

* [Go 1.4](http://golang.org/doc/install)
* [Godep](https://github.com/tools/godep)

Then, you can run `godep go install` to build the `openshift-cucumber` executable (in `$GOPATH/bin`)

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