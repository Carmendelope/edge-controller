# edge-controller
Egde controller for the Service Net

# Introduction

The Edge Controller is a component that will be deployed on the user premises and it is in charge of managing a collection of
agents. In terms of how that component will be deployed, we envision a VM deployment that could potentially be transformed
into an appliance.

# Sync and Async operations

The EIC uses both sync and async communication with the management cluster. The following table describes
the different types of operations and how those are performed

 | Operation  | Type | Local persistence | Description |
 | ------------- | ------------- |------------- |------------- |
 | Agent Join  | SYNC | No | An agent wants to join the EIC |
 | Agent Start | ASYNC | Yes | An agent starts |
 | Agent Callback | ASYNC | Yes | An agent sends a callback from an operation |
