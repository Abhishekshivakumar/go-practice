

Submitting a new feature request ensures that the requested feature aligns with the project’s scope and objectives. The requester must present a strong case to convince the approvers of the feature's merits. Providing comprehensive details and context is crucial for approval.

Steps for Submitting a New Feature Request

1. Confirm Task Visibility

Before proceeding, determine whether your task is public or protected.

Obtain confirmation from the approvers regarding the task’s visibility.

2. Copy an Existing Task

Select an appropriate task from the linked repository.

Copy the task and rename the metadata field accordingly.

Review the parameters and remove any that are unnecessary for your use case.

Utilize reserved parameters managed by Lightspeed (parameters prefixed with ls-<name>).

Refer to the Lightspeed Reserved Parameters documentation for available options.

3. Cleanup and Define Task Steps

In the newly created task YAML file:

Define all necessary steps required for your use case.

Minimize external script dependencies by embedding scripts within your Docker image.

Ensure clarity and readability in the task definition.

4. Creating a Docker Image

If your task requires a new Docker image, refer to the Docker Image Creation Guide in the current suites.

5. Testing Your Task

Deploy and test your task in dev-cluster namespace or ls-cluster namespace:

Dev-cluster namespace: Used for development and testing.

LS-cluster namespace: Dedicated for Tekton tasks.

To deploy in dev-cluster namespace, you must be a member of the dev-test-project in Lightspeed. Request the developer role via the Lightspeed Dev UI.

6. Applying the Task in LS-Cluster Namespace

Create a branch in the ls-pipeline-factory-config repository.

Commit the changes and deploy the task in the dev namespace.

A cron job runs every 10 minutes to apply the latest changes.

7. Testing the Pipeline

Navigate to the dev-test-project in Lightspeed.

Create a new project or use an existing one for testing your task.

If no pipeline exists, create a dedicated project and reference your task in the pipeline.yaml.

Commit your changes to the source repository.

Trigger a pipeline run via the OpenShift UI:

Click the blue spin button to initiate the pipeline run.

If successful, the pipeline run will complete without issues.

If errors occur, review the logs, identify the issue, and apply necessary fixes.