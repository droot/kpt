# Namespace provisioning example using UI

In this guide, we will use the package orchestration UI to:

- Register blueprint and deployment repositories
- Create a `kpt` package from scratch in blueprint repo
- Create a deployable instance of the `kpt` package
- Deploy the package in a kubernetes cluster

## Prerequisites

Before we begin, ensure that you are able to access the UI for `package orchestraction`.
The UI is provided as `Backstage` plugin and can be accessed at `[BACKSTAGE_BASE_URL]/config-as-data`.

*If you don't have a UI installed, follow the steps in [UI installation guide](guides/porch-ui-installation.md)
to install the UI.*

## Repository Registration

Ignore this section if you already have `blueprint` and `deployment` repo registered
with `porch`.

### Registering blueprint repository

To register a blueprint repo, start by clicking `Register Repository` button in
the upper right corner. Enter the following details in the Register Repository flow:

- In Repository Details, the Git URL is the clone url from your repository (i.e. https://github.com/platkrm/chris-blueprints.git).
Branch and directory can be left blank. Branch will default to `main` and
directory will default to `/`.
- In Repository Authentication, you’ll need to use the GitHub Personal Access Token
(unless your repository allows for unauthenticated writes).
Create a new secret for the Personal Access Token as directed by the flow.
- In Repository Content, be sure to select Blueprints.

Once the repository is registered, use the breadcrumbs (upper left) to navigate
back to the Repositories view.

### Registering deployment repository

To register a deployment repo, start by clicking `Register Repository` button in
the upper right corner. Enter the following details in the Register Repository flow:

- In Repository Details, the `git URL` is the clone url from your repository (i.e. https://github.com/platkrm/chris-deployments.git).
`Branch` and `directory` can be left blank. Branch will default to `main` and
directory will default to `/`.
- In Repository Authentication, either create a new secret, or optionally,
select the same secret in the Authentication Secret dropdown you created for
registering blueprint repo.
- In Repository Content, be sure to select `Deployments`.
- In Upstream Repository, select from the already registered Blueprint repositories.

Once the repository is created, use the breadcrumbs (upper left) to navigate
back to the Repositories view.

## Creating a Blueprint from scratch

Now that we have our repositories registered, we are ready to create our first
blueprint using the UI.

- Click on `Blueprints` tab to see the blueprint repositories. Select a blueprint
repo where you want to add new blueprint by clicking on it.
- Clicking on it will take you to a new screen where you can see the packages/blueprints
that exist in the selected repository. If this is a new repo, list will be empty.
- Click on `Add Blueprint` button in the upper right corner to create a new blueprint.
- In `Add Blueprint` interface, complete the setup by entering required information.
Figure below shows a screenshot of `Add Blueprint` interface.
![add-blueprint](/static/images/porch-ui/add-blueprint.png)
- After completing the above flow, you’ll be taken to your newly created blueprint
(see screenshot below). Here you will have a chance to add, edit, and remove
resources and functions.
![list-blueprint](/static/images/porch-ui/list-blueprints.png)
- Clicking on any of the resources on the table (currently the `Kptfile` and `ConfigMap`)
will show the resource viewer dialog where you can see quick information for each
resource, and optionally view the yaml for the resource.
- On the blueprint (see screenshot in Step 5), click ‘Edit’ to be able to edit the
blueprint. After clicking Edit, you should see this screen where you have an option
to add new resources.
![add-resources](/static/images/porch-ui/add-resources-blueprint.png)
- Using the ‘Add Resource’ button, add a new Namespace resource. Name the
namespace `example`.
- Click on Kptfile resource, add a new mutator
  - Search for ‘namespace’ and select ‘set-namespace’ with the latest version
    available for selection.
  - Select ‘kptfile.kpt.dev’ in the config map dropdown
  - By setting both of these values, anytime the blueprint is rendered
    (for instance, on save or when a deployable instance of the blueprint is created),
    the namespace will be set to the name of the package.
- Using the ‘Add Resource’ button, add a new Role Binding resource
  - Name the resource ‘app-admin’
  - In Role Reference, select ‘cluster role’ and set ‘app-admin’ as the name
  - Click Add Subject, and in the newly added subject, select ‘Group’ and set the
    name to ‘example.admin@bigco.com’.
- Using the ‘Add Resource’ button, add a new Apply Replacements resource.
  - Name the resource ‘update-rolebinding’
  - Using the yaml view, paste in the following yaml

```yaml
apiVersion: fn.kpt.dev/v1alpha1
kind: ApplyReplacements
metadata:
  name: update-rolebinding
  annotations:
    config.kubernetes.io/local-config: "true"
replacements:
- source:
    kind: ConfigMap
    name: kptfile.kpt.dev
    fieldPath: data.name
  targets:
  - select:
      name: app-admin
      kind: RoleBinding
    fieldPaths:
    - subjects.[kind=Group].name
    options:
      delimiter: '.'
      index: 0
```

- Using the ‘Add Resource’ button, add a new Resource Quota resource
  - Name the resource ‘base’
  - Set Max CPU Requests to 40 and Max Memory Requests to 40G
- Click on the Kptfile resource, add a new mutator
  - Search for ‘replacements’ and select `apply-replacements` with the latest
    version available for selection
  - Select the ApplyReplacements update-rolebinding local config object
- After you are done with the above, you should see the following
![save-blueprint](/static/images/porch-ui/save-blueprint.png)
- Clicking `Save` will save the resources, apply the mutator, and take you
  back to the blueprint screen you started on. Note that the namespace has
  been updated on the resources from the ‘set-namespace’ mutator.
![propose-blueprint](/static/images/porch-ui/propose-blueprint.png)
- Click on the individual resources to see the first class editors.
- Click Propose to propose the blueprint (button will change to Approve)
- Click Approve to approve the blueprint
- Using the breadcrumbs, click back to view your blueprints repository - here
  you should see the blueprint you just created has been finalized.
![finalized-blueprint](/static/images/porch-ui/finalized-blueprint.png)

So, with that, we created a blueprint from scratch and published it in blueprint
repo. You should be able to see the blueprint in the `git` repo as well.

## Create deployable instance of a blueprint

in this section, we will walk through the steps of creating a deployable instance
of a blueprint.

- Assuming you are on the repository screen, click on `Deployments` tab to list
deployment repositories (as shown below). Select a deployment repository by
clicking on it.
![list-deployment-repos](/static/images/porch-ui/list-deployment-repos.png)
- Here is the view for listing all deployments in the deployment repo.
![deployments-list](/static/images/porch-ui/list-deployment-repos.png)
- Click on the `Upstream Repository` link to navigate to the blueprints
 repository and here you should see `basens` blueprint.
![blueprint-list](/static/images/porch-ui/list-blueprints.png)
- Click on the `basens` blueprint.
![blueprint-show](/static/images/porch-ui/show-blueprint.png)
- Click the ‘Deploy’ button in the upper right corner to take you to the
  `Add Deployment` flow. Create the new deployment with the name ‘backend’.
![add-deployment](/static/images/porch-ui/add-deployment.png)
- Complete the flow and the package will be added to your deployments
 repository. Note that the namespace across all the resources has been updated
 to the name of the package.
![added-deployment](/static/images/porch-ui/added-deployment.png)
- Using the breadcrumbs, click back the deployments view to see your new
 deployment is added in Draft status.
![draft-deployment-screenshot](/static/images/porch-ui/draft-deployment.png)
- Click into the backend deployment and move the deployment to Proposed then
  Published by approving the deployment.
  Optionally, before moving the deployment to Published  If you wish too,
  you can make changes to the deployment by adding/removing/updating resources.
![draft-deployment](/static/images/porch-ui/draft-deployment.png)
- Once the deployment is published, click `Create Sync` to have the `Config Sync`
  sync the deployment to the kubernetes cluster. After a few seconds, you’ll see
  a Sync status update in the upper right hand corner.
![synced-deployment](/static/images/porch-ui/synced-deployment.png)
- If you navigate back to the `deployment` repo, you will see `sync` status next
  to each deployement instance.
![synced-deployment-screenshot](/static/images/porch-ui/synced-deployment-list.png)

So, this completes our end to end workflow of creating a blueprint from scratch
and deploying it to a kubernetes cluster using package orchestraction UI.
