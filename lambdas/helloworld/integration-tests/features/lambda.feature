Feature: Our Lambda is a friendly and simple thing that greets us with aplomb

  Scenario: Lambda responds to a basic request
    Given the Lambda is deployed
    When a request is made to the API endpoint
    Then a greeting is returned
