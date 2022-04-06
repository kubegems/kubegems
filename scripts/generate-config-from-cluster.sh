#! /bin/sh

kubectl -n kubegems expose service kubegems-gitea-http --name=kubegems-gitea --type=NodePort
kubectl -n kubegems expose service kubegems-redis-master --name=kubegems-redis --type=NodePort
kubectl -n kubegems expose service mysql --name=kubegems-mysql --type=NodePort
kubectl -n kubegems expose service kubegems-argocd-server --name=kubegems-argocd --type=NodePort --port=80 --target-port=server
kubectl -n kubegems patch svc kubegems-msgbus --patch='{"spec":{"type":"NodePort"}}'

argocdPort=$(kubectl -n kubegems get svc kubegems-argocd -o jsonpath='{.spec.ports[0].nodePort}')
argocdPassword=$(kubectl -n kubegems get secrets argocd-initial-admin-secret -ogo-template='{{ .data.password | base64decode }}')
redisPort=$(kubectl -n kubegems get svc kubegems-redis -o jsonpath='{.spec.ports[0].nodePort}')
redisPassword=$(kubectl -n kubegems get secrets kubegems-redis -ogo-template='{{ index .data "redis-password" | base64decode }}')
giteaPort=$(kubectl -n kubegems get svc kubegems-gitea -o jsonpath='{.spec.ports[0].nodePort}')
giteaPassword=$(kubectl -n kubegems get secrets kubegems-config -ogo-template='{{ .data.GIT_PASSWORD | base64decode }}')
mysqlPort=$(kubectl -n kubegems get svc kubegems-mysql -o jsonpath='{.spec.ports[0].nodePort}')
mysqlPassword=$(kubectl -n kubegems get secrets mysql -ogo-template='{{ index .data "mysql-root-password" | base64decode }}')
msgbusPort=$(kubectl -n kubegems get svc kubegems-msgbus -o jsonpath='{.spec.ports[0].nodePort}')
nodeAddress=$(kubectl get node -ojsonpath='{.items[0].status.addresses[0].address}')

cat <<EOF | tee config/config.yaml
mysql:
  addr: ${nodeAddress}:${mysqlPort}
  password: ${mysqlPassword}
redis:
  addr: ${nodeAddress}:${redisPort}
  password: ${redisPassword}
git:
  addr: http://${nodeAddress}:${giteaPort}
  password: ${giteaPassword}
argo:
  addr: http://${nodeAddress}:${argocdPort}
  password: ${argocdPassword}
msgbus:
  addr: http://${nodeAddress}:${msgbusPort}
EOF
