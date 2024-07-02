Feature: Small Download
    As a service
    I want to download a small file as a single fragment
    So that I can verify the integrity of the download

    Scenario: Downloading a small file as a single fragment
        Given I have received a download event for a small file
        When I download the file
        Then the file should be downloaded as a single fragment
