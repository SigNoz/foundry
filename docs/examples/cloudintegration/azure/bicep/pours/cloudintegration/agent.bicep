param location string
param containerEnvName string
param customRoleId string
param signozApiKey string
param signozApiUrl string
param signozIngestionUrl string
param signozIngestionKey string
param signozIntegrationAccountId string
param signozIntegrationAgentVersion string
param integrationConfigSyncPeriodSeconds string = '60'

// 1. Container App Environment (The shared network space)
resource containerAppEnv 'Microsoft.App/managedEnvironments@2023-05-01' = {
  name: containerEnvName
  location: location
  properties: {
    appLogsConfiguration: {}
  }
}

// 2. The Agent's Managed Identity
resource agentIdentity 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' = {
  name: 'signoz-integration-agent-identity'
  location: location
}

// 3. The Go Agent Container App
resource agentApp 'Microsoft.App/containerApps@2023-05-01' = {
  name: 'signoz-integration-agent'
  location: location
  identity: {
    type: 'UserAssigned'
    userAssignedIdentities: { '${agentIdentity.id}': {} }
  }
  properties: {
    managedEnvironmentId: containerAppEnv.id
    template: {
      containers: [
        {
          name: 'agent'
          image: 'signoz/cloud-integration-agent:${signozIntegrationAgentVersion}'
          resources: {
            cpu: any('0.25')
            memory: '0.5Gi'
          }
          env: [
            { name: 'CLOUD_PROVIDER', value: 'azure' }
            { name: 'COLLECTOR_ROLE_ID', value: customRoleId }
            { name: 'SIGNOZ_API_URL', value: signozApiUrl }
            { name: 'SIGNOZ_API_KEY', value: signozApiKey }
            { name: 'INGESTION_URL', value: signozIngestionUrl }
            { name: 'INGESTION_KEY', value: signozIngestionKey }
            { name: 'SIGNOZ_CLOUD_INTEGRATION_ACCOUNT_ID', value: signozIntegrationAccountId }
            { name: 'INTEGRATION_CONFIG_SYNC_PERIOD_SECONDS', value: integrationConfigSyncPeriodSeconds }
            { name: 'AZURE_CLIENT_ID', value: agentIdentity.properties.clientId }
            { name: 'AZURE_CONTAINER_APP_ENV_ID', value: containerAppEnv.id }
          ]
        }
      ]
    }
  }
}

output agentPrincipalId string = agentIdentity.properties.principalId
