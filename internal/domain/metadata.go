package domain

// MetadataPrefix namespaces every key foundry stamps onto, or reads from,
// something it generates: labels on a workload, annotations a user writes, tags
// on a cloud resource. Declaring it once is what keeps the three families in one
// namespace even though nothing compares them.
//
// Keys are one segment deep by convention, foundry.signoz.io/managed-by rather
// than foundry.signoz.io/ecs/cluster-id, so the namespace stays flat and
// greppable.
const MetadataPrefix = "foundry.signoz.io/"
