@loggedInFromEnvVars
Feature: Hello OpenShift from Docker image
  This feature test the hello-openshift application,
	using the hello-openshift template that deploys a docker image from docker hub.
	It use the OpenShift instance referenced by the OPENSHIFT_HOST env var,
	and the credentials provided by the OPENSHIFT_USER / OPENSHIFT_PASSWD env vars.

	Scenario: Create Project
		Given There is no project "hello-openshift-cucumber-test"
		When I create a new project "hello-openshift-cucumber-test"
		Then I should have a project "hello-openshift-cucumber-test"

	Scenario: Create template hello-openshift
		Given My current project is "hello-openshift-cucumber-test"
		And I have a file "examples/hello-openshift.yml"
		When I create a new template for the file "examples/hello-openshift.yml"
		Then I should have a template "hello-openshift" with 3 objects and 2 parameters

	Scenario: Create application hello
		Given My current project is "hello-openshift-cucumber-test"
		And I have a template "hello-openshift"
		When I create a new application based on the template "hello-openshift" with parameters "APPLICATION_NAME=hello"
		Then I should have a deploymentconfig "hello"
		And I should have a service "hello"
		And I should have a route "hello"

	Scenario: Check deployment
		Given My current project is "hello-openshift-cucumber-test"
		And I have a deploymentconfig "hello"
		When the deploymentconfig "hello" has at least 1 deployment
		Then the latest deployment of "hello" should succeed in less than "2m"

	Scenario: Check http endpoint availability
		Given My current project is "hello-openshift-cucumber-test"
		When I have a successful deployment of "hello"
		Then I can access the application through the route "hello"

	Scenario: Remove the project
		Given My current project is "hello-openshift-cucumber-test"
		And I can access the application through the route "hello"
		When I delete the project "hello-openshift-cucumber-test"
		Then I should not have a project "hello-openshift-cucumber-test"
