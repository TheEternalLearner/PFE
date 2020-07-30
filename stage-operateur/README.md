# README : 

### to deploy project

#### if controller has changed
-  make docker-build
-  kind --name lami load docker-image controller:latest
- helm repo add jetstack https://charts.jetstack.io
- helm repo update
- helm install \
  cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --version v0.15.1 \
  --set installCRDs=true
- kubectl rollout restart -n stage-operateur-system deployment stage-operateur-controller-manager 
- make deploy

#### if manifests have to be refreshed

- make deploy