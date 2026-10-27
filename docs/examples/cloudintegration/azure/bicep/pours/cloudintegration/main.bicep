targetScope = 'subscription'

param location string
param signozApiKey string
param signozApiUrl string
param signozIngestionUrl string
param signozIngestionKey string
param signozIntegrationAccountId string
param signozIntegrationAgentVersion string
param rgName string = 'signoz-integration-rg'
param containerEnvName string = 'signoz-integration-agent-env'

// 1. Create the Resource Group where the Agent lives
resource rg 'Microsoft.Resources/resourceGroups@2021-04-01' = {
  name: rgName
  location: location
}

// 2. Define the Custom Role (Subscription Level)
// This gives the Agent permission to manage the Collector and Policies
resource customRole 'Microsoft.Authorization/roleDefinitions@2022-04-01' = {
  name: guid('signoz-automation-role', subscription().id)
  properties: {
    roleName: 'SigNozAutomationAgent'
    description: 'Permissions for SigNoz Agent to manage monitoring and workload stacks'
    permissions: [
      {
        actions: [
          'Microsoft.Resources/deployments/*' // To manage the Collector Stack
          'Microsoft.Resources/deploymentStacks/*' // To manage the Collector Stack
          'Microsoft.Authorization/roleAssignments/write'
          'Microsoft.Insights/diagnosticSettings/*'
          'Microsoft.App/containerApps/*' // To deploy/update the collector container app
          'Microsoft.App/managedEnvironments/join/action' // To deploy container apps into an existing managed environment
          'Microsoft.ManagedIdentity/userAssignedIdentities/assign/action' // To assign managed identity to container apps
          'Microsoft.EventHub/namespaces/*' // To create/manage Event Hub namespaces and event hubs for log routing
          '*/read'
        ]
        dataActions: [
          'Microsoft.EventHub/namespaces/messages/receive/action'
        ]
      }
    ]
    assignableScopes: [subscription().id]
  }
}

// 3. Deploy the Agent Infrastructure
module agentResources './agent.bicep' = {
  name: 'signoz-agent-infra'
  scope: rg
  params: {
    location: location
    containerEnvName: containerEnvName
    customRoleId: customRole.id
    signozApiKey: signozApiKey
    signozApiUrl: signozApiUrl
    signozIngestionUrl: signozIngestionUrl
    signozIngestionKey: signozIngestionKey
    signozIntegrationAccountId: signozIntegrationAccountId
    signozIntegrationAgentVersion: signozIntegrationAgentVersion
  }
}

var agentIdentityName = 'signoz-integration-agent-identity'

// 4. Assign the Custom Role to the Identity at Subscription Scope
// The condition restricts the agent's roleAssignments/write action so it can only
// assign the Azure Event Hubs Data Receiver role (used for OTel collector → Event Hub auth).
// This prevents privilege escalation even if the agent identity is compromised.
resource roleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(subscription().id, rgName, agentIdentityName, customRole.id)
  scope: subscription()
  properties: {
    principalId: agentResources.outputs.agentPrincipalId
    roleDefinitionId: customRole.id
    principalType: 'ServicePrincipal'
    conditionVersion: '2.0'
    condition: '''(
 !(ActionMatches{'Microsoft.Authorization/roleAssignments/write'})
 OR
 @Request[Microsoft.Authorization/roleAssignments:RoleDefinitionId] ForAnyOfAnyValues:GuidEquals {a638d3c7-ab3a-418d-83e6-5f17a39d4fde}
)'''
  }
}
