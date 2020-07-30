## Validating operation on resource quota

### Creation
- ne pas créer de resourceQuota si cela dépasse les limites de cpu et de mémoire du projet
- resourceQuota should not be created if the project global limits are exceeded, this should also prevent namespace creation

### Update

- resourceQuota should forbid resourceQuota updates if it exceeds global project limits

### Deletion

- the default resourceQuota cannot be deleted until the current namespace is terminating 
- other resourceQuotas that are not related to the project can be deleted

