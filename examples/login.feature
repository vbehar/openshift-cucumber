Feature: Login
	Test that we can login on OpenShift

	Scenario: Login locally with the default demo user
		Given I have a username "demo"
		And I have a password "demo"
		When I login on "https://localhost:8443"
		Then I should be logged in as user "demo"
		And I should have a token

	Scenario: Login using the env vars
		Given I have a username "$OPENSHIFT_USER"
		And I have a password "$OPENSHIFT_PASSWD"
		When I login on "$OPENSHIFT_HOST"
		Then I should be logged in as user "$OPENSHIFT_USER"
		And I should have a token
